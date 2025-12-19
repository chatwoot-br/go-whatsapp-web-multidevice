# Issue Investigation: Conversation Split Across Two Chats (@lid vs @s.whatsapp.net)

**Date**: 2025-12-19
**Severity**: Critical
**Status**: Open - Investigation Complete
**Upstream Issue**: https://github.com/aldinokemal/go-whatsapp-web-multidevice/issues/484
**Version Affected**: v7.10+
**Reporter**: Javierd

## Issue Summary

WhatsApp conversations are incorrectly split into two separate chats when dealing with new contacts. Incoming messages are stored with `@lid` (Linked ID) JID format, while outgoing messages create a separate chat with `@s.whatsapp.net` (phone number) JID format. Both represent the same user but appear as unrelated conversations in the Web UI and API.

### Symptoms

- Same contact appears as two different chats
- Incoming messages go to one chat (`@lid` format)
- Outgoing/reply messages go to another chat (`@s.whatsapp.net` format)
- ~260+ errors in 24 hours: `Panic recovered in middleware: chat with JID [number] not found`
- Problem persists after full Docker container/volume cleanup
- Occurs specifically with unknown/new contacts

### Example Case

```
Incoming conversation: 3161230196747@lid
Reply creates separate: 34698820XXX@s.whatsapp.net
```

Both JIDs represent the same WhatsApp user, but the system treats them as distinct.

## Impact

**Severity: CRITICAL**

- **Data Integrity**: Conversations fragmented across multiple chat entries
- **User Experience**: Unable to view complete conversation history
- **Integration Breaking**: Downstream systems (Chatwoot, etc.) receive inconsistent data
- **Production Reliability**: Prevents reliable production deployment

### Affected Operations

1. **Failing**: Incoming message storage with LID JID
2. **Failing**: Outgoing message storage creates duplicate chat
3. **Failing**: Chat history lookups return incomplete data
4. **Affected**: All webhook consumers receiving inconsistent JID formats

## Root Cause Analysis

### Technical Background: LID vs Phone Number JIDs

WhatsApp's multi-device protocol uses two JID formats:

| Format | Example | Description |
|--------|---------|-------------|
| `@s.whatsapp.net` | `34698820XXX@s.whatsapp.net` | Traditional phone number-based JID |
| `@lid` | `3161230196747@lid` | Linked ID - opaque identifier for multi-device |

The `@lid` format is an internal WhatsApp identifier that maps to a phone number. The mapping is stored in the `whatsmeow_lid_map` database table.

### The Problem Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INCOMING MESSAGE FLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  New Contact Sends Message                                                   │
│         │                                                                    │
│         ▼                                                                    │
│  WhatsApp delivers with @lid JID (e.g., 3161230196747@lid)                  │
│         │                                                                    │
│         ▼                                                                    │
│  NormalizeJIDFromLID() called                                               │
│         │                                                                    │
│         ▼                                                                    │
│  GetPNForLID() → FAILS (no mapping for new contact)                         │
│         │                                                                    │
│         ▼                                                                    │
│  Falls back to original @lid JID                                            │
│         │                                                                    │
│         ▼                                                                    │
│  Message stored with chat_jid = "3161230196747@lid"  ◄── CHAT #1            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         OUTGOING MESSAGE FLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Bot/User Replies to Contact                                                 │
│         │                                                                    │
│         ▼                                                                    │
│  API receives phone number: 34698820XXX                                      │
│         │                                                                    │
│         ▼                                                                    │
│  ValidateJidWithLogin() → ParseJID()                                         │
│         │                                                                    │
│         ▼                                                                    │
│  Creates JID: 34698820XXX@s.whatsapp.net                                    │
│         │                                                                    │
│         ▼                                                                    │
│  StoreSentMessageWithContext() called                                        │
│         │                                                                    │
│         ▼                                                                    │
│  Message stored with chat_jid = "34698820XXX@s.whatsapp.net"  ◄── CHAT #2   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

RESULT: Two separate chats for the same user!
```

### Why LID Mapping is Missing for New Contacts

According to whatsmeow documentation, LID-to-phone-number mappings are populated through:

1. **History Sync** - Only runs at initial WhatsApp connection
2. **LID Migration Sync Messages** - Protocol-level sync events
3. **GetUserInfo Calls** - Explicitly fetching user information
4. **Group Participant Lists** - When fetching group members
5. **Device Notifications** - When devices connect/disconnect

**For a completely new contact who messages first, none of these mechanisms have triggered yet.** The LID store has no entry for this contact.

### Affected Code Paths

| File | Line | Function | Issue |
|------|------|----------|-------|
| `src/infrastructure/whatsapp/init.go` | 97-124 | `NormalizeJIDFromLID` | Falls back to `@lid` when `GetPNForLID` fails |
| `src/infrastructure/chatstorage/sqlite_repository.go` | 523-600 | `CreateMessage` | Stores message with unresolved `@lid` JID |
| `src/infrastructure/chatstorage/sqlite_repository.go` | 651-722 | `StoreSentMessageWithContext` | Always creates `@s.whatsapp.net` JID |
| `src/pkg/utils/whatsapp.go` | 637-645 | `ValidateJidWithLogin` | Returns `@s.whatsapp.net` format only |
| `src/infrastructure/whatsapp/event_message.go` | 28-66 | `createMessagePayload` | Webhook may contain unresolved `@lid` |

### Code Analysis

**NormalizeJIDFromLID (`src/infrastructure/whatsapp/init.go:97-124`):**
```go
func NormalizeJIDFromLID(ctx context.Context, jid types.JID, client *whatsmeow.Client) types.JID {
    if jid.Server != "lid" {
        return jid
    }

    // Safety check
    if client == nil || client.Store == nil || client.Store.LIDs == nil {
        log.Warnf("Cannot resolve LID %s: client not available", jid.String())
        return jid  // ← FALLBACK: Returns @lid unchanged
    }

    pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
    if err != nil {
        log.Debugf("Failed to resolve LID %s to phone number: %v", jid.String(), err)
        return jid  // ← FALLBACK: Returns @lid unchanged
    }

    if !pn.IsEmpty() {
        return pn  // ← SUCCESS: Returns @s.whatsapp.net
    }

    return jid  // ← FALLBACK: Returns @lid unchanged
}
```

**StoreSentMessageWithContext (`src/infrastructure/chatstorage/sqlite_repository.go:651-722`):**
```go
func (r *SQLiteRepository) StoreSentMessageWithContext(...) error {
    // ...
    jid, err := types.ParseJID(recipientJID)  // recipientJID is phone number
    // ...

    // This normalizes, but for phone numbers it's already @s.whatsapp.net
    normalizedJID := whatsapp.NormalizeJIDFromLID(ctx, jid, client)
    chatJID := normalizedJID.String()  // Always @s.whatsapp.net for sent messages

    // Creates/updates chat with @s.whatsapp.net JID
    // Does NOT check for existing @lid chat!
    // ...
}
```

## Proposed Solutions

### Option 1: Proactive LID Resolution (Recommended)

**Concept:** When receiving an incoming message with an `@lid` JID that cannot be resolved, proactively call `client.GetUserInfo()` to fetch and store the LID-to-phone-number mapping.

**Implementation:**
```go
func NormalizeJIDFromLID(ctx context.Context, jid types.JID, client *whatsmeow.Client) types.JID {
    if jid.Server != "lid" {
        return jid
    }

    if client == nil || client.Store == nil || client.Store.LIDs == nil {
        return jid
    }

    // First attempt
    pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
    if err == nil && !pn.IsEmpty() {
        return pn
    }

    // NEW: Proactive resolution via GetUserInfo
    _, err = client.GetUserInfo(ctx, []types.JID{jid})
    if err != nil {
        log.Debugf("Failed to fetch user info for LID %s: %v", jid.String(), err)
        return jid
    }

    // GetUserInfo populates the LID store automatically - retry lookup
    pn, err = client.Store.LIDs.GetPNForLID(ctx, jid)
    if err == nil && !pn.IsEmpty() {
        log.Debugf("Resolved LID %s to PN %s after GetUserInfo", jid.String(), pn.String())
        return pn
    }

    return jid
}
```

**Pros:**
- Fixes the problem at the source
- All downstream code automatically uses correct JID
- Consistent `@s.whatsapp.net` format everywhere
- Webhook payloads will have resolved phone numbers

**Cons:**
- Adds latency to incoming message processing (network call)
- May hit WhatsApp rate limits with many new contacts
- `GetUserInfo` may fail for some edge cases

**Risk Mitigation:**
- Add caching to avoid repeated calls for the same LID
- Make the call async with a short timeout
- Log failures but don't block message processing

---

### Option 2: Chat JID Aliasing (Fallback Lookup)

**Concept:** When storing a sent message, check if there's an existing chat with the corresponding `@lid` format. If found, use that chat's JID instead.

**Implementation:**
```go
func (r *SQLiteRepository) StoreSentMessageWithContext(...) error {
    // ... existing code ...

    normalizedJID := whatsapp.NormalizeJIDFromLID(ctx, jid, client)
    chatJID := normalizedJID.String()

    // NEW: Check if chat exists, if not try @lid format
    existingChat, _ := r.GetChat(chatJID)
    if existingChat == nil {
        lidJID := whatsapp.GetLIDForPN(ctx, normalizedJID, client)
        if !lidJID.IsEmpty() {
            if lidChat, _ := r.GetChat(lidJID.String()); lidChat != nil {
                chatJID = lidJID.String()  // Use existing @lid chat
            }
        }
    }

    // ... rest of function using chatJID ...
}
```

**Pros:**
- No additional network calls for incoming messages
- Works with existing data
- No latency added to message receiving

**Cons:**
- Requires reverse LID lookup
- Two JID formats still exist in database
- More complex lookup logic

---

### Option 3: Bidirectional JID Alias Table

**Concept:** Maintain a separate alias table linking `@lid` and `@s.whatsapp.net` JIDs.

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS jid_aliases (
    lid_jid TEXT PRIMARY KEY,
    pn_jid TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pn_jid)
);
CREATE INDEX idx_jid_aliases_pn ON jid_aliases(pn_jid);
```

**Pros:**
- Works with existing data
- Flexible - handles any JID format discrepancy
- Self-healing as mappings are discovered

**Cons:**
- Additional database table and queries
- More complex lookup logic
- Doesn't fix root cause

---

### Option 4: Consistent LID Usage

**Concept:** When sending, check if we have an `@lid` for the recipient and use that format consistently.

**Implementation:**
```go
func (r *SQLiteRepository) StoreSentMessageWithContext(...) error {
    jid, err := types.ParseJID(recipientJID)

    // Try to get LID for this phone number
    if jid.Server == types.DefaultUserServer {
        if lid, err := client.Store.LIDs.GetLIDForPN(ctx, jid); err == nil && !lid.IsEmpty() {
            jid = lid  // Use LID format
        }
    }

    chatJID := jid.String()
    // ...
}
```

**Pros:**
- Simple implementation
- Consistent with WhatsApp's internal format

**Cons:**
- `@lid` is less human-readable
- May break existing integrations expecting phone numbers
- Requires LID to already exist

## Recommended Approach

**Hybrid Solution: Option 1 + Option 2 + Option 4 (LID-First Approach)**

This approach prioritizes consistency with WhatsApp's internal format by **always using `@lid` as the canonical JID format** for chat storage.

### Design Philosophy

WhatsApp's multi-device protocol uses LID (Linked ID) as the primary identifier internally. The `@lid` format is:
- **Stable**: Doesn't change if user changes phone number
- **Consistent**: What WhatsApp actually uses in the protocol
- **Future-proof**: WhatsApp is moving towards LID-based identification

### Solution Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INCOMING MESSAGE FLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Message arrives with @lid JID                                               │
│         │                                                                    │
│         ▼                                                                    │
│  [Option 1] Try GetPNForLID() to get phone number                           │
│         │                                                                    │
│         ├──► Success: Store PN↔LID mapping for future lookups               │
│         │                                                                    │
│         ▼                                                                    │
│  Store message with @lid JID (canonical format)                             │
│         │                                                                    │
│         ▼                                                                    │
│  Webhook: Include both "from_lid" and resolved "from" (phone)               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         OUTGOING MESSAGE FLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  API receives phone number: 34698820XXX                                      │
│         │                                                                    │
│         ▼                                                                    │
│  [Option 4] GetLIDForPN(34698820XXX) - lookup LID for this phone            │
│         │                                                                    │
│         ├──► Found LID: Use @lid format for storage                         │
│         │                                                                    │
│         ├──► Not found: [Option 1] Call GetUserInfo to fetch LID            │
│         │         │                                                          │
│         │         ├──► Success: Use @lid format                              │
│         │         │                                                          │
│         │         └──► Failed: [Option 2] Check existing @lid chats         │
│         │                                                                    │
│         ▼                                                                    │
│  Store message with @lid JID (same chat as incoming)                        │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

RESULT: Single chat using consistent @lid format!
```

### Implementation Components

#### 1. LID Resolver Service (New File)

**File: `src/infrastructure/whatsapp/lid_resolver.go`**

```go
package whatsapp

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// LIDResolver handles bidirectional LID ↔ PN resolution with caching
type LIDResolver struct {
	client       *whatsmeow.Client
	pendingLIDs  map[string]time.Time // Track recently failed lookups to avoid spam
	mu           sync.RWMutex
	cacheTTL     time.Duration
}

var (
	resolver     *LIDResolver
	resolverOnce sync.Once
)

// GetLIDResolver returns the singleton LID resolver instance
func GetLIDResolver() *LIDResolver {
	resolverOnce.Do(func() {
		resolver = &LIDResolver{
			pendingLIDs: make(map[string]time.Time),
			cacheTTL:    5 * time.Minute, // Don't retry failed lookups for 5 min
		}
	})
	resolver.client = cli // Update client reference
	return resolver
}

// ResolveToLID converts any JID to its @lid format if possible
// This is the primary function for ensuring consistent LID usage
func (r *LIDResolver) ResolveToLID(ctx context.Context, jid types.JID) types.JID {
	if r.client == nil || r.client.Store == nil || r.client.Store.LIDs == nil {
		return jid
	}

	// Already @lid format - return as-is
	if jid.Server == "lid" {
		return jid
	}

	// Only process @s.whatsapp.net JIDs
	if jid.Server != types.DefaultUserServer {
		return jid
	}

	// Check if we recently failed to resolve this JID
	r.mu.RLock()
	if lastAttempt, exists := r.pendingLIDs[jid.String()]; exists {
		if time.Since(lastAttempt) < r.cacheTTL {
			r.mu.RUnlock()
			return jid // Skip - recently failed
		}
	}
	r.mu.RUnlock()

	// Try to get LID from store
	lid, err := r.client.Store.LIDs.GetLIDForPN(ctx, jid)
	if err == nil && !lid.IsEmpty() {
		logrus.Debugf("Resolved PN %s to LID %s from store", jid.String(), lid.String())
		return lid
	}

	// [Option 1] Proactive resolution via GetUserInfo
	lid = r.proactiveResolve(ctx, jid)
	if !lid.IsEmpty() {
		return lid
	}

	// Mark as failed to avoid repeated attempts
	r.mu.Lock()
	r.pendingLIDs[jid.String()] = time.Now()
	r.mu.Unlock()

	return jid
}

// ResolveToPNForWebhook converts @lid to phone number for external systems
// Webhooks should receive phone numbers for compatibility
func (r *LIDResolver) ResolveToPNForWebhook(ctx context.Context, jid types.JID) (pnJID types.JID, lidJID types.JID) {
	if r.client == nil || r.client.Store == nil || r.client.Store.LIDs == nil {
		return jid, types.EmptyJID
	}

	// Not @lid - return as-is
	if jid.Server != "lid" {
		return jid, types.EmptyJID
	}

	lidJID = jid

	// Try to resolve to phone number
	pn, err := r.client.Store.LIDs.GetPNForLID(ctx, jid)
	if err == nil && !pn.IsEmpty() {
		return pn, lidJID
	}

	// [Option 1] Proactive resolution
	pn = r.proactiveResolveLIDToPN(ctx, jid)
	if !pn.IsEmpty() {
		return pn, lidJID
	}

	// Return original @lid if resolution fails
	return jid, lidJID
}

// proactiveResolve attempts to resolve PN to LID via GetUserInfo
func (r *LIDResolver) proactiveResolve(ctx context.Context, pnJID types.JID) types.JID {
	resolveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := r.client.GetUserInfo(resolveCtx, []types.JID{pnJID})
	if err != nil {
		logrus.Debugf("GetUserInfo failed for PN %s: %v", pnJID.String(), err)
		return types.EmptyJID
	}

	// Retry lookup after GetUserInfo (it should have populated the store)
	lid, err := r.client.Store.LIDs.GetLIDForPN(ctx, pnJID)
	if err == nil && !lid.IsEmpty() {
		logrus.Infof("Proactively resolved PN %s to LID %s", pnJID.String(), lid.String())
		return lid
	}

	return types.EmptyJID
}

// proactiveResolveLIDToPN attempts to resolve LID to PN via GetUserInfo
func (r *LIDResolver) proactiveResolveLIDToPN(ctx context.Context, lidJID types.JID) types.JID {
	resolveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := r.client.GetUserInfo(resolveCtx, []types.JID{lidJID})
	if err != nil {
		logrus.Debugf("GetUserInfo failed for LID %s: %v", lidJID.String(), err)
		return types.EmptyJID
	}

	pn, err := r.client.Store.LIDs.GetPNForLID(ctx, lidJID)
	if err == nil && !pn.IsEmpty() {
		logrus.Infof("Proactively resolved LID %s to PN %s", lidJID.String(), pn.String())
		return pn
	}

	return types.EmptyJID
}

// CleanupCache removes stale entries from the pending cache
func (r *LIDResolver) CleanupCache() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for jid, lastAttempt := range r.pendingLIDs {
		if now.Sub(lastAttempt) > r.cacheTTL {
			delete(r.pendingLIDs, jid)
		}
	}
}
```

#### 2. Modified CreateMessage (Incoming Messages)

**File: `src/infrastructure/chatstorage/sqlite_repository.go`**

```go
func (r *SQLiteRepository) CreateMessage(ctx context.Context, evt *events.Message) error {
	if evt == nil || evt.Message == nil {
		return nil
	}

	client := whatsapp.GetClient()
	resolver := whatsapp.GetLIDResolver()

	// For incoming messages, keep @lid format as canonical
	chatJID := evt.Info.Chat
	senderJID := evt.Info.Sender

	// If chat is @s.whatsapp.net (edge case), try to convert to @lid
	if chatJID.Server == types.DefaultUserServer {
		if resolved := resolver.ResolveToLID(ctx, chatJID); resolved.Server == "lid" {
			chatJID = resolved
		}
	}

	// Normalize sender similarly
	if senderJID.Server == types.DefaultUserServer {
		if resolved := resolver.ResolveToLID(ctx, senderJID); resolved.Server == "lid" {
			senderJID = resolved
		}
	}

	chatJIDStr := chatJID.String()
	senderJIDStr := senderJID.String()

	// Get chat name - resolve to PN for human-readable name
	pnJID, _ := resolver.ResolveToPNForWebhook(ctx, chatJID)
	chatName := r.GetChatNameWithPushName(pnJID, chatJIDStr, pnJID.User, evt.Info.PushName)

	// [Option 2] Check for existing chat with @s.whatsapp.net format and migrate
	if chatJID.Server == "lid" {
		r.migrateExistingPNChat(ctx, chatJID, pnJID)
	}

	// ... rest of existing CreateMessage logic using chatJIDStr ...
}

// migrateExistingPNChat migrates an existing @s.whatsapp.net chat to @lid format
func (r *SQLiteRepository) migrateExistingPNChat(ctx context.Context, lidJID, pnJID types.JID) {
	if pnJID.IsEmpty() || pnJID.Server != types.DefaultUserServer {
		return
	}

	pnChatJID := pnJID.String()
	lidChatJID := lidJID.String()

	// Check if there's an existing chat with the PN format
	existingPNChat, err := r.GetChat(pnChatJID)
	if err != nil || existingPNChat == nil {
		return // No migration needed
	}

	// Check if LID chat already exists
	existingLIDChat, _ := r.GetChat(lidChatJID)

	logrus.Infof("Migrating chat from PN %s to LID %s", pnChatJID, lidChatJID)

	tx, err := r.db.Begin()
	if err != nil {
		logrus.Errorf("Failed to begin migration transaction: %v", err)
		return
	}
	defer tx.Rollback()

	if existingLIDChat == nil {
		// Rename the PN chat to LID format
		_, err = tx.Exec("UPDATE chats SET jid = ? WHERE jid = ?", lidChatJID, pnChatJID)
		if err != nil {
			logrus.Errorf("Failed to rename chat: %v", err)
			return
		}
	}

	// Update all messages to use LID format
	_, err = tx.Exec("UPDATE messages SET chat_jid = ? WHERE chat_jid = ?", lidChatJID, pnChatJID)
	if err != nil {
		logrus.Errorf("Failed to update messages: %v", err)
		return
	}

	// Update sender JIDs in messages if they match the PN format
	_, err = tx.Exec("UPDATE messages SET sender = ? WHERE sender = ?", lidChatJID, pnChatJID)
	if err != nil {
		logrus.Errorf("Failed to update message senders: %v", err)
		return
	}

	// Delete the old PN chat if we merged into existing LID chat
	if existingLIDChat != nil {
		_, err = tx.Exec("DELETE FROM chats WHERE jid = ?", pnChatJID)
		if err != nil {
			logrus.Errorf("Failed to delete old PN chat: %v", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		logrus.Errorf("Failed to commit migration: %v", err)
		return
	}

	logrus.Infof("Successfully migrated chat from PN %s to LID %s", pnChatJID, lidChatJID)
}
```

#### 3. Modified StoreSentMessageWithContext (Outgoing Messages)

**File: `src/infrastructure/chatstorage/sqlite_repository.go`**

```go
func (r *SQLiteRepository) StoreSentMessageWithContext(ctx context.Context, messageID string,
	senderJID string, recipientJID string, content string, timestamp time.Time) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	jid, err := types.ParseJID(recipientJID)
	if err != nil {
		return fmt.Errorf("invalid JID format: %w", err)
	}

	client := whatsapp.GetClient()
	resolver := whatsapp.GetLIDResolver()

	// [Option 4] Always try to resolve to @lid format for consistency
	resolvedJID := resolver.ResolveToLID(ctx, jid)
	chatJID := resolvedJID.String()

	// [Option 2] If we couldn't get LID, check for existing @lid chat
	if resolvedJID.Server != "lid" {
		existingLIDChat := r.findExistingLIDChat(ctx, jid)
		if existingLIDChat != "" {
			chatJID = existingLIDChat
			logrus.Debugf("Using existing LID chat %s for PN %s", chatJID, jid.String())
		}
	}

	// Get chat name - use PN for human-readable name
	pnJID := jid
	if resolvedJID.Server == "lid" {
		pnJID, _ = resolver.ResolveToPNForWebhook(ctx, resolvedJID)
	}
	chatName := r.GetChatNameWithPushName(pnJID, chatJID, pnJID.User, "")

	// ... rest of existing logic ...
}

// findExistingLIDChat searches for an existing chat with @lid format for this phone number
func (r *SQLiteRepository) findExistingLIDChat(ctx context.Context, pnJID types.JID) string {
	client := whatsapp.GetClient()
	if client == nil || client.Store == nil || client.Store.LIDs == nil {
		return ""
	}

	// Try to get LID from store (might exist from previous messages)
	lid, err := client.Store.LIDs.GetLIDForPN(ctx, pnJID)
	if err != nil || lid.IsEmpty() {
		return ""
	}

	lidChatJID := lid.String()

	// Check if this LID chat exists in our database
	if existingChat, _ := r.GetChat(lidChatJID); existingChat != nil {
		return lidChatJID
	}

	return ""
}
```

#### 4. Modified Webhook Payload (External Compatibility)

**File: `src/infrastructure/whatsapp/event_message.go`**

```go
func createMessagePayload(ctx context.Context, evt *events.Message,
	downloadedMedia *DownloadedMedia) (map[string]any, error) {

	message := utils.BuildEventMessage(evt)
	waReaction := utils.BuildEventReaction(evt)
	forwarded := utils.BuildForwarded(evt)

	body := make(map[string]any)
	resolver := GetLIDResolver()

	// Always include raw IDs
	body["sender_id"] = evt.Info.Sender.User
	body["chat_id"] = evt.Info.Chat.User

	// Resolve sender for external systems - they need phone numbers
	senderPN, senderLID := resolver.ResolveToPNForWebhook(ctx, evt.Info.Sender)

	// Resolve chat JID similarly
	chatPN, chatLID := resolver.ResolveToPNForWebhook(ctx, evt.Info.Chat)

	// Primary "from" field should be phone number for compatibility
	if from := evt.Info.SourceString(); from != "" {
		fromUser, fromGroup := from, ""
		if strings.Contains(from, " in ") {
			parts := strings.Split(from, " in ")
			fromUser = parts[0]
			fromGroup = parts[1]
		}

		// Build resolved "from" with phone number
		if !senderPN.IsEmpty() && senderPN.Server == types.DefaultUserServer {
			if fromGroup != "" {
				body["from"] = fmt.Sprintf("%s in %s", senderPN.String(), fromGroup)
			} else {
				body["from"] = senderPN.String()
			}
		} else {
			body["from"] = from
		}

		// Always include LID if available
		if !senderLID.IsEmpty() {
			body["from_lid"] = senderLID.String()
		}
	}

	// Include resolved chat JID
	if !chatPN.IsEmpty() {
		body["chat_jid"] = chatPN.String()
	}
	if !chatLID.IsEmpty() {
		body["chat_lid"] = chatLID.String()
	}

	// ... rest of existing payload building ...
}
```

### Webhook Payload Format

With this approach, webhooks include **both** formats for maximum compatibility:

```json
{
  "from": "34698820XXX@s.whatsapp.net",
  "from_lid": "3161230196747@lid",
  "chat_jid": "34698820XXX@s.whatsapp.net",
  "chat_lid": "3161230196747@lid",
  "sender_id": "34698820XXX",
  "chat_id": "3161230196747",
  "message": {
    "id": "ABC123",
    "text": "Hello!"
  }
}
```

External systems can use:
- `from` / `chat_jid` - Human-readable phone number format
- `from_lid` / `chat_lid` - Internal LID format (stable identifier)

### Benefits

| Aspect | Benefit |
|--------|---------|
| **Consistency** | All chats use `@lid` format internally |
| **Stability** | LID doesn't change if user changes phone number |
| **WhatsApp Alignment** | Matches WhatsApp's internal protocol |
| **Self-healing** | Automatically migrates old `@s.whatsapp.net` chats |
| **Backward Compatible** | Webhooks still include phone numbers |
| **Future-proof** | Ready for WhatsApp's continued LID adoption |

### Trade-offs and Mitigations

| Concern | Mitigation |
|---------|------------|
| LID not human-readable | Chat `name` field stores readable name; webhook includes PN |
| Initial latency | `GetUserInfo` has 3s timeout; cached to avoid repeats |
| Migration complexity | Automatic, transparent migration on first message |
| Rate limits | Cache failed lookups for 5 minutes |

## Testing Plan

### Unit Tests

1. Test `NormalizeJIDFromLID` with:
   - Valid LID that can be resolved
   - Valid LID that cannot be resolved (new contact)
   - Non-LID JID (passthrough)
   - Nil client handling

2. Test `StoreSentMessageWithContext` with:
   - Existing chat with matching JID
   - No existing chat (should create new)
   - Existing chat with `@lid` format (should find and use)

### Integration Tests

1. Simulate new contact message flow:
   - Receive message with `@lid`
   - Send reply
   - Verify both messages in same chat

2. Simulate existing contact message flow:
   - Ensure no regression for resolved LIDs

### Manual Testing

1. Connect fresh WhatsApp account
2. Have new contact send message
3. Reply to message via API
4. Verify single chat in database and UI

## Related Documentation

- [whatsmeow LID handling](https://github.com/tulir/whatsmeow) - Upstream library
- [Webhook Payload Documentation](../webhook-payload.md) - JID format in webhooks
- [Chat Storage Architecture](../developer/architecture.md) - Database design

## External References

- **Upstream Issue**: https://github.com/aldinokemal/go-whatsapp-web-multidevice/issues/484
- **whatsmeow Repository**: https://github.com/tulir/whatsmeow
- **WhatsApp Multi-Device Protocol**: Internal WhatsApp documentation

## Action Items

### Phase 1: Core Implementation
- [ ] Create `src/infrastructure/whatsapp/lid_resolver.go` - LID Resolver service
- [ ] Implement `ResolveToLID()` - PN to LID resolution with caching
- [ ] Implement `ResolveToPNForWebhook()` - LID to PN for external systems
- [ ] Implement proactive resolution via `GetUserInfo`

### Phase 2: Storage Layer Updates
- [ ] Modify `CreateMessage()` to use LID as canonical format
- [ ] Implement `migrateExistingPNChat()` for automatic migration
- [ ] Modify `StoreSentMessageWithContext()` to resolve PN to LID
- [ ] Implement `findExistingLIDChat()` fallback lookup

### Phase 3: Webhook Compatibility
- [ ] Update `createMessagePayload()` to include both formats
- [ ] Add `from_lid` and `chat_lid` fields to webhook payload
- [ ] Update webhook documentation with new fields

### Phase 4: Testing
- [ ] Add unit tests for LIDResolver
- [ ] Add unit tests for chat migration logic
- [ ] Add integration tests for incoming/outgoing message flow
- [ ] Test with production-like data volume
- [ ] Test webhook payload format with downstream systems

### Phase 5: Deployment
- [ ] Deploy to staging environment
- [ ] Validate chat consistency in staging
- [ ] Monitor for 48 hours
- [ ] Deploy to production
- [ ] Monitor migration logs

---

**Investigation Author**: Claude Code
**Last Updated**: 2025-12-19
**Status**: Investigation Complete - Awaiting Implementation
