package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// NormalizeJIDFromLID converts @lid JIDs to their corresponding @s.whatsapp.net JIDs
// Returns the original JID if it's not an @lid or if LID lookup fails
func NormalizeJIDFromLID(ctx context.Context, jid types.JID, client *whatsmeow.Client) types.JID {
	// Only process @lid JIDs
	if jid.Server != "lid" {
		return jid
	}

	// Safety check
	if client == nil || client.Store == nil || client.Store.LIDs == nil {
		log.Warnf("Cannot resolve LID %s: client not available", jid.String())
		return jid
	}

	// Attempt to get the phone number for this LID
	pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
	if err != nil {
		log.Debugf("Failed to resolve LID %s to phone number: %v", jid.String(), err)
		return jid
	}

	// If we got a valid phone number, use it
	if !pn.IsEmpty() {
		log.Debugf("Resolved LID %s to phone number %s", jid.String(), pn.String())
		return pn
	}

	// Fallback to original JID
	return jid
}

// NormalizeJIDFromLIDWithContext converts @lid JIDs to @s.whatsapp.net JIDs using its
// own context with a 30-second timeout. Used from history-sync post-completion paths
// (deduplicateLIDChats, forwardHistorySyncCompleteToWebhook) where the originating
// event context may already be cancelled.
//
// TODO(slice-6): collapse with upstream's ResolveLIDToPhone — the no-context
// NormalizeJIDFromLID above overlaps with upstream's helper; this WithContext
// variant is the fork-unique surface that needs to survive.
func NormalizeJIDFromLIDWithContext(jid types.JID, client *whatsmeow.Client) types.JID {
	// Only process @lid JIDs
	if jid.Server != "lid" {
		return jid
	}

	// Safety check
	if client == nil || client.Store == nil || client.Store.LIDs == nil {
		log.Warnf("Cannot resolve LID %s: client not available", jid.String())
		return jid
	}

	// Create dedicated context with generous timeout
	// This prevents "context canceled" errors from short-lived event contexts
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt to get the phone number for this LID
	pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
	if err != nil {
		log.Debugf("Failed to resolve LID %s to phone number: %v", jid.String(), err)
		return jid
	}

	// If we got a valid phone number, use it
	if !pn.IsEmpty() {
		log.Debugf("Resolved LID %s to phone number %s", jid.String(), pn.String())
		return pn
	}

	// Fallback to original JID
	return jid
}
