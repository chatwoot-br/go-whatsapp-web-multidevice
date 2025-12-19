package whatsapp

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
)

// LIDResolver handles bidirectional resolution between LID and PN (phone number) JIDs.
// It provides caching for failed lookups to avoid repeated expensive operations.
// This fixes the LID-JID chat split issue (Issue #484).
type LIDResolver struct {
	pendingLIDs map[string]time.Time // cache for failed lookups
	mu          sync.RWMutex
	cacheTTL    time.Duration
}

var (
	lidResolverInstance *LIDResolver
	lidResolverOnce     sync.Once
)

// GetLIDResolver returns the singleton LIDResolver instance.
// The resolver uses the global WhatsApp client from init.go.
// It starts a background goroutine for cache cleanup.
func GetLIDResolver() *LIDResolver {
	lidResolverOnce.Do(func() {
		lidResolverInstance = &LIDResolver{
			pendingLIDs: make(map[string]time.Time),
			cacheTTL:    5 * time.Minute,
		}

		// Start background cleanup goroutine to prevent memory leaks
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()

			for range ticker.C {
				if lidResolverInstance != nil {
					lidResolverInstance.CleanupCache()
				}
			}
		}()
	})
	return lidResolverInstance
}

// ResolveToLID converts a phone number JID (@s.whatsapp.net) to its corresponding LID JID (@lid).
// Returns the original JID if:
// - It's already a @lid JID
// - It's not a @s.whatsapp.net JID
// - Resolution fails after proactive lookup
func (r *LIDResolver) ResolveToLID(ctx context.Context, jid types.JID) types.JID {
	// Return as-is if already @lid format
	if jid.Server == "lid" {
		return jid
	}

	// Only process @s.whatsapp.net JIDs
	if jid.Server != types.DefaultUserServer {
		return jid
	}

	client := GetClient()

	// Safety check
	if client == nil || client.Store == nil || client.Store.LIDs == nil {
		logrus.Debugf("[LIDResolver] Cannot resolve PN to LID %s: client not available", jid.String())
		return jid
	}

	// Check cache for recently failed lookups
	cacheKey := jid.String()
	r.mu.RLock()
	if failTime, exists := r.pendingLIDs[cacheKey]; exists {
		if time.Since(failTime) < r.cacheTTL {
			r.mu.RUnlock()
			logrus.Debugf("[LIDResolver] Skipping cached failed lookup for %s", jid.String())
			return jid
		}
	}
	r.mu.RUnlock()

	// Try GetLIDForPN first (local store lookup)
	lidJID, err := client.Store.LIDs.GetLIDForPN(ctx, jid)
	if err != nil {
		logrus.Debugf("[LIDResolver] GetLIDForPN error for %s: %v", jid.String(), err)
	} else if !lidJID.IsEmpty() {
		logrus.Debugf("[LIDResolver] Resolved PN %s to LID %s from store", jid.String(), lidJID.String())
		return lidJID
	}

	// If not found, try proactive resolution via GetUserInfo
	lidJID = r.proactiveResolve(ctx, jid)
	if !lidJID.IsEmpty() && lidJID.Server == "lid" {
		// Remove from failed cache since resolution succeeded
		r.mu.Lock()
		delete(r.pendingLIDs, cacheKey)
		r.mu.Unlock()
		logrus.Debugf("[LIDResolver] Proactively resolved PN %s to LID %s", jid.String(), lidJID.String())
		return lidJID
	}

	// Cache failed attempt for 5 minutes
	r.mu.Lock()
	r.pendingLIDs[cacheKey] = time.Now()
	r.mu.Unlock()
	logrus.Debugf("[LIDResolver] Failed to resolve PN %s to LID, cached as pending", jid.String())

	return jid
}

// ResolveToPNForWebhook converts a LID JID (@lid) to its corresponding phone number JID (@s.whatsapp.net).
// Returns both the PN JID and the original LID JID for webhook compatibility.
// If the input is not a @lid JID, returns the original JID for pnJID and an empty JID for lidJID.
func (r *LIDResolver) ResolveToPNForWebhook(ctx context.Context, jid types.JID) (pnJID types.JID, lidJID types.JID) {
	// Return original if not @lid
	if jid.Server != "lid" {
		return jid, types.EmptyJID
	}

	lidJID = jid

	client := GetClient()

	// Safety check
	if client == nil || client.Store == nil || client.Store.LIDs == nil {
		logrus.Debugf("[LIDResolver] Cannot resolve LID to PN %s: client not available", jid.String())
		return jid, lidJID
	}

	// Try GetPNForLID first (local store lookup)
	pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
	if err != nil {
		logrus.Debugf("[LIDResolver] GetPNForLID error for %s: %v", jid.String(), err)
	} else if !pn.IsEmpty() {
		logrus.Debugf("[LIDResolver] Resolved LID %s to PN %s from store", jid.String(), pn.String())
		return pn, lidJID
	}

	// If not found, try proactive resolution
	pn = r.proactiveResolveLIDToPN(ctx, jid)
	if !pn.IsEmpty() && pn.Server == types.DefaultUserServer {
		logrus.Debugf("[LIDResolver] Proactively resolved LID %s to PN %s", jid.String(), pn.String())
		return pn, lidJID
	}

	logrus.Debugf("[LIDResolver] Failed to resolve LID %s to PN, returning original", jid.String())
	return jid, lidJID
}

// GetLIDForPN is a convenience method to get the LID for a phone number.
// Returns empty JID and error if not found.
func (r *LIDResolver) GetLIDForPN(ctx context.Context, pn types.JID) (types.JID, error) {
	client := GetClient()
	if client == nil || client.Store == nil || client.Store.LIDs == nil {
		return types.EmptyJID, nil
	}
	return client.Store.LIDs.GetLIDForPN(ctx, pn)
}

// proactiveResolve uses GetUserInfo to populate the LID store and retry resolution.
// This is called when the local store doesn't have a mapping for the phone number.
func (r *LIDResolver) proactiveResolve(ctx context.Context, pnJID types.JID) types.JID {
	client := GetClient()
	if client == nil {
		return types.EmptyJID
	}

	// Use 3 second timeout for proactive resolution
	resolveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Call GetUserInfo to populate LID store
	// This triggers the server to send back LID information
	_, err := client.GetUserInfo(resolveCtx, []types.JID{pnJID})
	if err != nil {
		logrus.Debugf("[LIDResolver] GetUserInfo failed for PN %s: %v", pnJID.String(), err)
		return types.EmptyJID
	}

	// Retry GetLIDForPN after GetUserInfo (may have populated the store)
	// Use resolveCtx to respect the timeout budget
	lidJID, err := client.Store.LIDs.GetLIDForPN(resolveCtx, pnJID)
	if err != nil {
		logrus.Debugf("[LIDResolver] GetLIDForPN retry error for %s: %v", pnJID.String(), err)
		return types.EmptyJID
	}

	if !lidJID.IsEmpty() {
		logrus.Infof("[LIDResolver] Proactive resolution succeeded: PN %s -> LID %s", pnJID.String(), lidJID.String())
		return lidJID
	}

	return types.EmptyJID
}

// proactiveResolveLIDToPN attempts to resolve a LID to PN using server queries.
// This is called when the local store doesn't have a mapping for the LID.
func (r *LIDResolver) proactiveResolveLIDToPN(ctx context.Context, lidJID types.JID) types.JID {
	client := GetClient()
	if client == nil {
		return types.EmptyJID
	}

	// Use 3 second timeout for proactive resolution
	resolveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// For LID to PN resolution, we can try GetUserInfo with the LID
	// This may trigger the server to send back the PN information
	_, err := client.GetUserInfo(resolveCtx, []types.JID{lidJID})
	if err != nil {
		logrus.Debugf("[LIDResolver] GetUserInfo failed for LID %s: %v", lidJID.String(), err)
		return types.EmptyJID
	}

	// Retry GetPNForLID after GetUserInfo (may have populated the store)
	// Use resolveCtx to respect the timeout budget
	pnJID, err := client.Store.LIDs.GetPNForLID(resolveCtx, lidJID)
	if err != nil {
		logrus.Debugf("[LIDResolver] GetPNForLID retry error for %s: %v", lidJID.String(), err)
		return types.EmptyJID
	}

	if !pnJID.IsEmpty() {
		logrus.Infof("[LIDResolver] Proactive resolution succeeded: LID %s -> PN %s", lidJID.String(), pnJID.String())
		return pnJID
	}

	return types.EmptyJID
}

// CleanupCache removes stale entries from the pendingLIDs cache.
// Entries older than cacheTTL are removed.
func (r *LIDResolver) CleanupCache() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	removed := 0
	for key, failTime := range r.pendingLIDs {
		if now.Sub(failTime) >= r.cacheTTL {
			delete(r.pendingLIDs, key)
			removed++
		}
	}
	if removed > 0 {
		logrus.Debugf("[LIDResolver] Cleaned up %d stale cache entries", removed)
	}
}

// GetCacheSize returns the current number of entries in the pending cache.
// Useful for monitoring and debugging.
func (r *LIDResolver) GetCacheSize() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.pendingLIDs)
}

// ResetCache clears all entries from the pending cache.
// This can be called when the WhatsApp connection is reset.
func (r *LIDResolver) ResetCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pendingLIDs = make(map[string]time.Time)
	logrus.Info("[LIDResolver] Cache reset")
}
