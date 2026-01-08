# Plan: Fix WhatsApp LID Message Delivery Issue

## Problem
Messages sent to phone number `5532988659372` are delivered to the wrong contact on WhatsApp. This happens because go-whatsapp sends messages using phone number JIDs (`phone@s.whatsapp.net`) instead of LID JIDs (`number@lid`) when a LID mapping exists.

## Root Cause
- **Incoming messages**: Correctly resolve LID → phone using `GetPNForLID()` via `NormalizeJIDFromLID()`
- **Outgoing messages**: Only use phone JID, never check for existing LID mapping using `GetLIDForPN()`

---

## Initial Solution (Commit 9b947bc) - PROBLEMATIC

The initial approach modified `ValidateJidWithLogin()` to return LID instead of phone JID when a mapping exists. This was intended as a "centralized fix" that would apply to all send functions automatically.

### Why This Approach Fails

**`ValidateJidWithLogin` is used in 40+ places**, not just for sending messages:

| Category | Functions | Problem with LID |
|----------|-----------|------------------|
| **Send Operations** | SendText, SendImage, SendVideo, etc. | ✅ Need LID for correct delivery |
| **Message Operations** | DownloadMedia, MarkAsRead, ReactMessage, etc. | ❌ Compare JIDs with stored messages (stored as phone JID) |
| **Chat Operations** | PinChat, ArchiveChat, SetDisappearingTimer | ❌ App state patches indexed by phone JID |
| **Group Operations** | All group functions | ⚪ Use group JIDs, not affected |
| **User Operations** | GetUserInfo, CheckPhones | ❌ Expect phone JID format |

### Observed Failure

```
DEBU[1365] Using LID 206206362738938@lid for phone number 5511960425142@s.whatsapp.net
ERRO[1365] Panic recovered in middleware: message 3EB0C873E7180962C19B2D does not belong to chat 206206362738938@lid
```

**Flow causing the panic:**
1. DownloadMedia receives phone number `5511960425142@s.whatsapp.net`
2. `ValidateJidWithLogin` returns LID `206206362738938@lid`
3. Message stored with `ChatJID = 5511960425142@s.whatsapp.net`
4. Comparison fails: `5511960425142@s.whatsapp.net != 206206362738938@lid`
5. Error converted to panic via `PanicIfNeeded`

---

## Revised Solution: Separate Function for Send Operations

### Principle
**Explicit is better than implicit.** Create a dedicated function for LID resolution that's only used where actually needed (message sending), keeping `ValidateJidWithLogin` unchanged for all other operations.

### Implementation

#### 1. Revert `ValidateJidWithLogin` to original behavior

```go
// ValidateJidWithLogin validates JID with login check (returns phone JID)
func ValidateJidWithLogin(client *whatsmeow.Client, jid string) (types.JID, error) {
    MustLogin(client)

    if config.WhatsappAccountValidation && !IsOnWhatsapp(client, jid) {
        return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Phone %s is not on whatsapp", jid))
    }

    return ParseJID(jid)
}
```

#### 2. Create new function `ResolveJIDForSend` in `src/pkg/utils/whatsapp.go`

```go
// ResolveJIDForSend resolves a JID for sending messages, using LID when available.
// This ensures messages are delivered to the correct contact even if their phone number changed.
// Use this ONLY for client.SendMessage operations, not for comparisons or app state.
func ResolveJIDForSend(ctx context.Context, client *whatsmeow.Client, phoneJID types.JID) types.JID {
    // Only resolve LID for individual users, not groups or other types
    if phoneJID.Server != types.DefaultUserServer {
        return phoneJID
    }

    // Safety check
    if client == nil || client.Store == nil || client.Store.LIDs == nil {
        return phoneJID
    }

    // Attempt to get the LID for this phone number
    lid, err := client.Store.LIDs.GetLIDForPN(ctx, phoneJID)
    if err != nil {
        logrus.Debugf("No LID mapping for %s: %v", phoneJID.String(), err)
        return phoneJID
    }

    if !lid.IsEmpty() {
        logrus.Debugf("Resolved phone %s to LID %s for sending", phoneJID.String(), lid.String())
        return lid
    }

    return phoneJID
}
```

#### 3. Update send operations in `src/usecase/send.go`

Replace the pattern:
```go
dataWaRecipient, err := utils.ValidateJidWithLogin(client, request.BaseRequest.Phone)
// ... build message ...
ts, err := client.SendMessage(ctx, dataWaRecipient, msg)
```

With:
```go
dataWaRecipient, err := utils.ValidateJidWithLogin(client, request.BaseRequest.Phone)
// ... build message ...
sendJID := utils.ResolveJIDForSend(ctx, client, dataWaRecipient)
ts, err := client.SendMessage(ctx, sendJID, msg)
```

### Files to Modify

| File | Changes |
|------|---------|
| `src/pkg/utils/whatsapp.go` | Revert `ValidateJidWithLogin`, add `ResolveJIDForSend` |
| `src/usecase/send.go` | Use `ResolveJIDForSend` before `client.SendMessage` calls |

### Functions Requiring `ResolveJIDForSend`

Only functions that call `client.SendMessage`:

1. `SendText` (send.go:55)
2. `SendImage` (send.go via sendWithMedia)
3. `SendFile` (send.go via sendWithMedia)
4. `SendVideo` (send.go via sendWithMedia)
5. `SendAudio` (send.go via sendWithMedia)
6. `SendContact` (send.go)
7. `SendLink` (send.go)
8. `SendLocation` (send.go)
9. `SendPoll` (send.go)
10. `SendSticker` (send.go)
11. `ReactMessage` (message.go:95)
12. `RevokeMessage` (message.go:120)
13. `UpdateMessage` (message.go:186)

### Functions That Should NOT Change

These use `ValidateJidWithLogin` but don't need LID:

- `DownloadMedia` - compares with stored message JID
- `MarkAsRead` - marks read status
- `DeleteMessage` - app state patch
- `StarMessage` - app state patch
- `PinChat` - app state patch
- `ArchiveChat` - app state patch
- `SetDisappearingTimer` - sets timer
- All group operations - use group JIDs
- All user operations - query user info

---

## Why This Approach Is Better

| Aspect | Initial Approach | Revised Approach |
|--------|------------------|------------------|
| **Scope** | Changes behavior for 40+ call sites | Changes only ~13 send operations |
| **Risk** | High - breaks comparisons, app state | Low - targeted change |
| **Intent** | Implicit (magic behavior change) | Explicit (function name indicates purpose) |
| **Debugging** | Hard to trace why LID appears | Clear: `ResolveJIDForSend` shows intent |
| **Backward Compatible** | No - breaks existing behavior | Yes - validation unchanged |

---

## Verification

### 1. Build the project
```bash
cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice/src
go build ./...
```

### 2. Run tests
```bash
go test ./...
```

### 3. Manual Testing

**Test Case 1: Send message to contact with known LID**
1. Start go-whatsapp with debug logging
2. Send message to contact `5532988659372`
3. Check logs for: `Resolved phone X to LID Y for sending`
4. Verify message arrives at correct contact

**Test Case 2: Send message to new contact (no LID)**
1. Send message to a contact you've never messaged before
2. Check logs for: `No LID mapping for X`
3. Verify message delivers correctly

**Test Case 3: Download media (regression test)**
1. Receive a media message
2. Call DownloadMedia API
3. Verify no panic, media downloads correctly

**Test Case 4: Chat operations (regression test)**
1. Pin a chat
2. Archive a chat
3. Verify both operations succeed without errors

---

## Alternative Approaches Considered

### A. Post-hoc normalization
Normalize LID back to phone JID where comparisons happen.

**Rejected because:**
- Spreads complexity across codebase
- Easy to miss places that need normalization
- Doesn't clearly indicate intent

### B. Store LID alongside phone JID
Update storage schema to store both formats.

**Rejected because:**
- Requires database migration
- Significant effort for limited benefit
- Doesn't solve app state issues

### C. Let whatsmeow handle it
Check if whatsmeow already handles LID internally.

**Investigation result:**
- whatsmeow's `SendMessage` accepts any JID
- Server-side routing depends on the JID provided
- We must provide LID for correct routing when known

---

## Implementation Checklist

- [x] Revert `ValidateJidWithLogin` in `src/pkg/utils/whatsapp.go`
- [x] Add `ResolveJIDForSend` function
- [x] Update all `client.SendMessage` calls in `send.go` (via `wrapSendMessage`)
- [x] Update message operations that send (ReactMessage, RevokeMessage, UpdateMessage)
- [x] Update `auto_reply.go`
- [x] Build and verify no compilation errors
- [x] Run tests
- [ ] Manual testing with debug logs
- [ ] Verify DownloadMedia works (regression)
- [ ] Verify PinChat/ArchiveChat work (regression)
