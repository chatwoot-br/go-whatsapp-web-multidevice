# ISSUE-004: History Sync LID Resolution Context Canceled Causes Duplicate Chats

## Status: Fixed

## Summary
During history sync, LID-to-phone-number resolution fails with "context canceled" for some conversations, causing messages to be stored with `@lid` JIDs. When the same contact is later resolved successfully (or sends new real-time messages), a separate chat entry is created with the `@s.whatsapp.net` JID, resulting in duplicate contacts in the Chat List.

## Environment
- **go-whatsapp-web-multidevice**: v8.1.0+5
- **Test scenario**: History sync for contact with LID `215946727821336@lid` (phone: `556796707788`)

## Problem Description

### Observed Behavior
In the Chat List UI, the same contact appears twice:

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

### Expected Behavior
All messages for the same contact should be stored under a single, normalized JID (preferably `@s.whatsapp.net` format) regardless of when LID resolution succeeds.

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

### 3. No Deduplication Mechanism

**File**: `src/infrastructure/chatstorage/sqlite_repository.go`

The database uses `(jid, device_id)` as the primary key:
```sql
PRIMARY KEY (jid, device_id)
```

When the same contact is stored with two different JID formats, both entries persist as separate chats with no mechanism to merge them.

### Data Flow

```
History Sync Event (batch 1)
    │
    ▼
processConversationMessages() ─── ctx gets canceled
    │
    ▼
NormalizeJIDFromLID(ctx, jid, client)
    │
    ├── ctx canceled → return "215946727821336@lid"
    │                          │
    │                          ▼
    │                    Chat stored as "215946727821336@lid" (15 messages)
    │
History Sync Event (batch 2)
    │
    ▼
processConversationMessages() ─── ctx still active
    │
    ▼
NormalizeJIDFromLID(ctx, jid, client)
    │
    └── ctx active → return "556796707788@s.whatsapp.net"
                              │
                              ▼
                        Chat stored as "556796707788@s.whatsapp.net" (141 messages)

Result: TWO separate chat entries for the SAME contact
```

## Proposed Fix

### Option 1: Dedicated Context for LID Resolution (Recommended - Quick Fix)

**File**: `src/infrastructure/whatsapp/history_sync.go:172-174`

Create a separate, longer-lived context for LID resolution:

```go
// Before:
jid = NormalizeJIDFromLID(ctx, jid, client)

// After:
lidCtx, lidCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer lidCancel()
jid = NormalizeJIDFromLID(lidCtx, jid, client)
```

### Option 2: Add Retry Logic to LID Resolution

**File**: `src/infrastructure/whatsapp/jid_utils.go`

Add retry with exponential backoff:
```go
func NormalizeJIDFromLIDWithRetry(jid types.JID, client *whatsmeow.Client) types.JID {
    if jid.Server != "lid" {
        return jid
    }

    for attempts := 0; attempts < 3; attempts++ {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
        cancel()

        if err == nil && !pn.IsEmpty() {
            return pn
        }
        time.Sleep(time.Duration(attempts+1) * 100 * time.Millisecond)
    }
    return jid
}
```

### Option 3: Chat Deduplication/Merging

**File**: `src/infrastructure/chatstorage/sqlite_repository.go`

Add a method to merge duplicate chats when LID→phone mapping is discovered:
```go
func (r *SQLiteRepository) MergeLIDChat(deviceID, lidJID, phoneJID string) error {
    tx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Update all messages from lidJID to phoneJID
    _, err = tx.Exec(`
        UPDATE messages SET chat_jid = ?
        WHERE chat_jid = ? AND device_id = ?
    `, phoneJID, lidJID, deviceID)
    if err != nil {
        return err
    }

    // Delete the LID-based chat entry
    _, err = tx.Exec(`
        DELETE FROM chats WHERE jid = ? AND device_id = ?
    `, lidJID, deviceID)
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

### Option 4: Data Migration for Existing Duplicates

Create a one-time migration to fix existing duplicate chats:
```go
func (r *SQLiteRepository) FixDuplicateLIDChats(client *whatsmeow.Client) error {
    // Find all chats ending with @lid
    rows, err := r.db.Query(`SELECT device_id, jid FROM chats WHERE jid LIKE '%@lid'`)
    // For each, resolve LID to phone number
    // If phone-based chat exists, merge messages and delete LID chat
}
```

## Impact

### Current Impact
- Users see duplicate contacts in Chat List
- Message history is split across two chat entries
- Contact names may differ (LID-based shows as "Other" type)
- Confusion when viewing conversation history

### After Fix
- Single chat entry per contact
- Unified message history
- Consistent contact type and naming

## Testing

1. Clear chat storage database
2. Connect a WhatsApp account
3. Trigger history sync
4. Verify no "context canceled" errors for LID resolution
5. Verify each contact has only one chat entry
6. Verify contacts with `@lid` JIDs are normalized to `@s.whatsapp.net`

## Related Files
- `src/infrastructure/whatsapp/history_sync.go` - History sync processing
- `src/infrastructure/whatsapp/jid_utils.go` - LID resolution function
- `src/infrastructure/whatsapp/event_handler.go` - Event context handling
- `src/infrastructure/chatstorage/sqlite_repository.go` - Chat storage

## Related Issues
- ISSUE-002: LID chat_id not normalized (webhook side, fixed)
- This issue: Internal chat storage duplication (separate issue)

## Timeline
- **2026-01-18**: Issue identified from Chat List duplicate investigation
- **2026-01-18**: Root cause analysis completed - context cancellation during history sync
- **2026-01-18**: Fix options documented
- **2026-01-18**: Fix implemented

## Fix Implementation

### Changes Made

1. **New Function: `NormalizeJIDFromLIDWithContext`** (`src/infrastructure/whatsapp/jid_utils.go`)
   - Creates dedicated 30-second context for LID resolution
   - Prevents "context canceled" errors from event context lifecycle
   - Does not require parent context parameter

2. **History Sync Updates** (`src/infrastructure/whatsapp/history_sync.go`)
   - All `NormalizeJIDFromLID` calls replaced with `NormalizeJIDFromLIDWithContext`
   - Added `deduplicateLIDChats` function for post-sync cleanup
   - Deduplication runs after push name application during sync completion

3. **Chat Merge Capability** (`src/infrastructure/chatstorage/sqlite_repository.go`)
   - Added `MergeLIDChat` method to merge LID chats into phone chats
   - Added `GetLIDChats` to find all LID-based chats for a device
   - Merge preserves chat metadata (name, timestamp, ephemeral settings)

### Commits
- `feat(whatsapp): add NormalizeJIDFromLIDWithContext with dedicated timeout`
- `fix(history-sync): use dedicated context for LID resolution`
- `feat(chatstorage): add MergeLIDChat for deduplicating chats`
- `feat(history-sync): add post-sync LID chat deduplication`
- `fix(history-sync): use dedicated context for all LID resolutions`
