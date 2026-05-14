# Fix History Sync LID Duplicate Chats Implementation Plan

> **Status: COMPLETED** - All tasks implemented on 2026-01-18

**Goal:** Fix duplicate chat entries caused by LID resolution failures during history sync, ensuring each contact has a single chat entry.

**Architecture:** Two-layer fix: (1) Use dedicated long-lived context for LID resolution during history sync to prevent "context canceled" errors, (2) Add chat merge functionality to deduplicate existing LID-based chats when phone mapping is discovered.

**Tech Stack:** Go 1.21+, SQLite, whatsmeow library, testify for assertions

**Related Issue:** [ISSUE-004](../issues/ISSUE-004-history-sync-lid-context-canceled-duplicate-chats.md)

---

## Implementation Summary

| Task | Description | Status | Commit |
|------|-------------|--------|--------|
| 1 | Add NormalizeJIDFromLIDWithContext Function | DONE | `5b7591e` |
| 2 | Update History Sync to Use New LID Function | DONE | `0cad324` |
| 3 | Add MergeLIDChat Repository Method | DONE | `89e9405` |
| 4 | Simplify NormalizeJIDFromLIDWithContext | SKIPPED | N/A (already simple) |
| 5 | Add Post-Sync Deduplication Step | DONE | `80c7c81` |
| 6 | Update processPushNames to Use New Context | DONE | `14e4545` |
| 7 | Update Issue Documentation | DONE | `f5e591c`, `07c8333` |
| 8 | Final Integration Test | DONE | N/A |

### Implementation Notes

- **Task 3**: Required updating 3 interface implementations (not 2 as planned):
  - `SQLiteRepository` (main implementation)
  - `DeviceRepository` (wrapper in chatstorage package)
  - `deviceChatStorage` (wrapper in whatsapp package)
- **Task 3**: Code quality review caught a transaction consistency bug where `StoreChat` was called outside the transaction. Fixed by using `tx.Exec()` directly.
- **Task 4**: Skipped because the function was already implemented in the simplified form in Task 1 (no merge trigger needed).

---

## Task 1: Add NormalizeJIDFromLIDWithContext Function

**Status:** DONE | **Commit:** `5b7591e`

**Files:**
- Modify: `src/infrastructure/whatsapp/jid_utils.go`
- Create: `src/infrastructure/whatsapp/jid_utils_test.go`

**Implementation:**

```go
// NormalizeJIDFromLIDWithContext converts @lid JIDs to @s.whatsapp.net JIDs
// Uses its own context with 30-second timeout to avoid event context cancellation issues
// Returns the original JID if it's not an @lid or if LID lookup fails
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
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Attempt to get the phone number for this LID
    pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
    if err != nil {
        log.Debugf("Failed to resolve LID %s to phone number: %v", jid.String(), err)
        return jid
    }

    if !pn.IsEmpty() {
        log.Debugf("Resolved LID %s to phone number %s", jid.String(), pn.String())
        return pn
    }

    return jid
}
```

**Tests:** 2 tests for non-LID passthrough and nil client handling.

---

## Task 2: Update History Sync to Use New LID Function

**Status:** DONE | **Commit:** `0cad324`

**Files:**
- Modify: `src/infrastructure/whatsapp/history_sync.go`

**Changes:**
- Line 174: `jid = NormalizeJIDFromLIDWithContext(jid, client)`
- Line 244: `senderJID = NormalizeJIDFromLIDWithContext(senderJID, client)`

---

## Task 3: Add MergeLIDChat Repository Method

**Status:** DONE | **Commit:** `89e9405`

**Files:**
- Modify: `src/domains/chatstorage/interfaces.go`
- Modify: `src/infrastructure/chatstorage/sqlite_repository.go`
- Modify: `src/infrastructure/chatstorage/device_repository.go` (additional)
- Modify: `src/infrastructure/whatsapp/chatstorage_wrapper.go` (additional)

**Interface additions:**
```go
// LID deduplication
MergeLIDChat(deviceID, lidJID, phoneJID string) error
GetLIDChats(deviceID string) ([]*Chat, error)
```

**Note:** During implementation, discovered that 3 implementations of `IChatStorageRepository` needed updating, not just the 2 originally planned.

**Note:** Code review caught that `StoreChat` was being called outside the transaction. Fixed by replacing with direct `tx.Exec()` UPDATE.

---

## Task 4: Simplify NormalizeJIDFromLIDWithContext

**Status:** SKIPPED

**Reason:** The function was implemented in the simplified form from the start in Task 1. No merge trigger was ever added, so there was nothing to simplify.

---

## Task 5: Add Post-Sync Deduplication Step

**Status:** DONE | **Commit:** `80c7c81`

**Files:**
- Modify: `src/infrastructure/whatsapp/history_sync.go`

**Changes:**
- Added `deduplicateLIDChats()` function (lines 603-651)
- Called after `applyCachedPushNamesToChats` in `scheduleHistorySyncWebhook` (line 105)

---

## Task 6: Update processPushNames to Use New Context

**Status:** DONE | **Commit:** `14e4545`

**Files:**
- Modify: `src/infrastructure/whatsapp/history_sync.go`

**Changes:** Updated 5 remaining `NormalizeJIDFromLID` calls:
- Line 339: `processOnDemandHistorySync`
- Line 383: `forwardOnDemandMessageToWebhook` (chat JID)
- Line 393: `forwardOnDemandMessageToWebhook` (participant)
- Line 481: `processPushNames`
- Line 683: `forwardHistorySyncCompleteToWebhook`

---

## Task 7: Update Issue Documentation

**Status:** DONE | **Commits:** `f5e591c`, `07c8333`

**Files:**
- Modify: `docs/issues/ISSUE-004-history-sync-lid-context-canceled-duplicate-chats.md`

**Changes:**
- Updated status from "Open" to "Fixed"
- Added "Fix Implementation" section with all changes
- Restructured document to lead with the fix
- Added commit SHAs, data flow diagram, verification steps

---

## Task 8: Final Integration Test

**Status:** DONE

**Results:**
- Build: PASS
- Tests: All pass (5 tests in whatsapp package)
- Code quality: Approved by final review

**Manual verification steps:**
1. Clear chat storage: Delete `storages/chatstorage.db`
2. Start the application: `cd src && go run . rest`
3. Connect a WhatsApp account
4. Wait for history sync to complete
5. Check logs for:
   - No "context canceled" errors for LID resolution
   - "Deduplicated N LID-based chats" message (if applicable)
6. Verify Chat List has no duplicate contacts

---

## Files Changed Summary

| File | Change |
|------|--------|
| `src/infrastructure/whatsapp/jid_utils.go` | Add `NormalizeJIDFromLIDWithContext` |
| `src/infrastructure/whatsapp/jid_utils_test.go` | Add tests for new function |
| `src/infrastructure/whatsapp/history_sync.go` | Use new function (8 calls), add `deduplicateLIDChats` |
| `src/domains/chatstorage/interfaces.go` | Add `MergeLIDChat`, `GetLIDChats` |
| `src/infrastructure/chatstorage/sqlite_repository.go` | Implement merge methods |
| `src/infrastructure/chatstorage/device_repository.go` | Wrapper implementation |
| `src/infrastructure/whatsapp/chatstorage_wrapper.go` | Wrapper implementation |
| `docs/issues/ISSUE-004-*.md` | Update status to Fixed |
