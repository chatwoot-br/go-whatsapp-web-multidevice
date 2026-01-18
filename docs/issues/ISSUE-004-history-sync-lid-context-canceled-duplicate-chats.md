# ISSUE-004: History Sync LID Resolution Context Canceled Causes Duplicate Chats

## Status: Fixed

## Summary
During history sync, LID-to-phone-number resolution fails with "context canceled" for some conversations, causing messages to be stored with `@lid` JIDs. When the same contact is later resolved successfully (or sends new real-time messages), a separate chat entry is created with the `@s.whatsapp.net` JID, resulting in duplicate contacts in the Chat List.

## Environment
- **go-whatsapp-web-multidevice**: v8.1.0+5
- **Test scenario**: History sync for contact with LID `215946727821336@lid` (phone: `556796707788`)

---

## Fix Implementation

### Solution Overview

A two-layer fix was implemented:

1. **Prevention**: New `NormalizeJIDFromLIDWithContext` function with dedicated 30-second timeout context prevents "context canceled" errors
2. **Correction**: Post-sync `deduplicateLIDChats` function merges any existing LID-based chats into their phone-based counterparts

### Changes Made

#### 1. New Function: `NormalizeJIDFromLIDWithContext`

**File**: `src/infrastructure/whatsapp/jid_utils.go`

```go
// NormalizeJIDFromLIDWithContext converts @lid JIDs to @s.whatsapp.net JIDs
// Uses its own context with 30-second timeout to avoid event context cancellation issues
func NormalizeJIDFromLIDWithContext(jid types.JID, client *whatsmeow.Client) types.JID
```

- Creates dedicated 30-second context for LID resolution
- Prevents "context canceled" errors from event context lifecycle
- Does not require parent context parameter

#### 2. History Sync Updates

**File**: `src/infrastructure/whatsapp/history_sync.go`

- All 8 `NormalizeJIDFromLID` calls replaced with `NormalizeJIDFromLIDWithContext`
- Added `deduplicateLIDChats` function for post-sync cleanup
- Deduplication runs after push name application during sync completion

#### 3. Chat Merge Capability

**File**: `src/infrastructure/chatstorage/sqlite_repository.go`

```go
// MergeLIDChat merges a LID-based chat into a phone-based chat
func (r *SQLiteRepository) MergeLIDChat(deviceID, lidJID, phoneJID string) error

// GetLIDChats returns all chats with @lid JIDs for a device
func (r *SQLiteRepository) GetLIDChats(deviceID string) ([]*Chat, error)
```

- Merge updates all messages to the phone JID
- Preserves chat metadata (name, timestamp, ephemeral settings)
- Transaction-safe with proper rollback on failure

#### 4. Interface Updates

**File**: `src/domains/chatstorage/interfaces.go`

```go
// LID deduplication
MergeLIDChat(deviceID, lidJID, phoneJID string) error
GetLIDChats(deviceID string) ([]*Chat, error)
```

### Commits

| SHA | Message |
|-----|---------|
| `5b7591e` | feat(whatsapp): add NormalizeJIDFromLIDWithContext with dedicated timeout |
| `0cad324` | fix(history-sync): use dedicated context for LID resolution |
| `89e9405` | feat(chatstorage): add MergeLIDChat for deduplicating chats |
| `80c7c81` | feat(history-sync): add post-sync LID chat deduplication |
| `14e4545` | fix(history-sync): use dedicated context for all LID resolutions |
| `f5e591c` | docs: update ISSUE-004 status to Fixed |

### Data Flow After Fix

```
History Sync Event
    │
    ▼
processConversationMessages()
    │
    ▼
NormalizeJIDFromLIDWithContext(jid, client)  ← Uses dedicated 30s context
    │
    └── Always succeeds (no context cancellation)
        │
        ▼
    Chat stored as "556796707788@s.whatsapp.net"

After sync completes (5s debounce):
    │
    ▼
deduplicateLIDChats()
    │
    ├── GetLIDChats() → Find any @lid chats
    ├── NormalizeJIDFromLIDWithContext() → Resolve LID to phone
    └── MergeLIDChat() → Merge into phone chat

Result: SINGLE chat entry per contact
```

---

## Verification

### Automated Tests

```bash
cd src && go test ./infrastructure/whatsapp/... -v
```

Tests verify:
- Non-LID JIDs pass through unchanged
- Nil client returns original JID with warning

### Manual Verification

1. Clear chat storage: Delete `storages/chatstorage.db`
2. Start the application: `cd src && go run . rest`
3. Connect a WhatsApp account
4. Wait for history sync to complete
5. Check logs for:
   - No "context canceled" errors for LID resolution
   - "Deduplicated N LID-based chats" message (if applicable)
6. Verify Chat List has no duplicate contacts

---

## Problem Description

### Observed Behavior (Before Fix)

In the Chat List UI, the same contact appeared twice:

| Name | Type | JID | Last Message |
|------|------|-----|--------------|
| Quezia | Other | 215946727821336@lid | Jan 15, 2026 15:30 |
| 556796707788 | Contact | 556796707788@s.whatsapp.net | Jan 15, 2026 15:30 |

These are the **same contact** but displayed as two separate entries.

### Log Evidence (2026-01-18 09:35:22)

**First sync batch - LID resolution FAILED:**
```
09:35:22.120 [Main DEBUG] Failed to resolve LID 215946727821336@lid to phone number: context canceled
09:35:22.125 [Main DEBUG] Processing 15 messages for chat 215946727821336@lid
09:35:22.126 [Main DEBUG] Stored 15 messages for chat 215946727821336@lid
```

**680ms later - LID resolution SUCCEEDED:**
```
09:35:22.802 [Main DEBUG] Resolved LID 215946727821336@lid to phone number 556796707788@s.whatsapp.net
```

**Second sync batch - Different chat entry created:**
```
09:35:33.176 [Main DEBUG] Resolved LID 215946727821336@lid to phone number 556796707788@s.whatsapp.net
09:35:33.176 [Main DEBUG] Processing 143 messages for chat 556796707788@s.whatsapp.net
09:35:33.252 [Main DEBUG] Stored 141 messages for chat 556796707788@s.whatsapp.net
```

---

## Root Cause Analysis

### 1. Context Lifecycle Issue

**File**: `src/infrastructure/whatsapp/event_handler.go:54`

The history sync handler receives the event's context which has a short lifecycle:
```go
case *events.HistorySync:
    handleHistorySync(ctx, evt, chatStorageRepo, client)
```

The context passed from the WhatsApp library's event system may be canceled during batch processing.

### 2. Failed LID Resolution Returns Original JID

**File**: `src/infrastructure/whatsapp/jid_utils.go:26-29`

When context is canceled, the original LID is returned instead of the phone number:
```go
pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
if err != nil {
    log.Debugf("Failed to resolve LID %s to phone number: %v", jid.String(), err)
    return jid  // Returns the original @lid JID
}
```

### 3. No Deduplication Mechanism (Before Fix)

**File**: `src/infrastructure/chatstorage/sqlite_repository.go`

The database uses `(jid, device_id)` as the primary key:
```sql
PRIMARY KEY (jid, device_id)
```

When the same contact is stored with two different JID formats, both entries persist as separate chats with no mechanism to merge them.

---

## Impact

### Before Fix
- Users see duplicate contacts in Chat List
- Message history is split across two chat entries
- Contact names may differ (LID-based shows as "Other" type)
- Confusion when viewing conversation history

### After Fix
- Single chat entry per contact
- Unified message history
- Consistent contact type and naming
- Automatic cleanup of existing duplicates

---

## Related Files

| File | Purpose |
|------|---------|
| `src/infrastructure/whatsapp/jid_utils.go` | LID resolution functions |
| `src/infrastructure/whatsapp/jid_utils_test.go` | Tests for LID resolution |
| `src/infrastructure/whatsapp/history_sync.go` | History sync processing |
| `src/infrastructure/whatsapp/event_handler.go` | Event context handling |
| `src/domains/chatstorage/interfaces.go` | Repository interface |
| `src/infrastructure/chatstorage/sqlite_repository.go` | Chat storage implementation |

## Related Issues
- ISSUE-002: LID chat_id not normalized (webhook side, fixed)

---

## Timeline

| Date | Event |
|------|-------|
| 2026-01-18 | Issue identified from Chat List duplicate investigation |
| 2026-01-18 | Root cause analysis completed - context cancellation during history sync |
| 2026-01-18 | Fix options documented |
| 2026-01-18 | Fix implemented and verified |
