# Plan: Fix WhatsApp LID Message Delivery Issue

## Problem
Messages sent to phone number `5532988659372` are delivered to the wrong contact on WhatsApp. This happens because go-whatsapp sends messages using phone number JIDs (`phone@s.whatsapp.net`) instead of LID JIDs (`number@lid`) when a LID mapping exists.

## Root Cause
- **Incoming messages**: Correctly resolve LID → phone using `GetPNForLID()` via `NormalizeJIDFromLID()`
- **Outgoing messages**: Only use phone JID, never check for existing LID mapping using `GetLIDForPN()`

## Solution
Modify `ValidateJidWithLogin()` in `src/pkg/utils/whatsapp.go` to check for LID mapping before returning the JID. This is a **centralized fix** that will apply to all 13 send functions automatically.

## Files to Modify

### 1. `src/pkg/utils/whatsapp.go` (Primary Change)

**Current function (lines 654-663):**
```go
func ValidateJidWithLogin(client *whatsmeow.Client, jid string) (types.JID, error) {
    MustLogin(client)
    if config.WhatsappAccountValidation && !IsOnWhatsapp(client, jid) {
        return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Phone %s is not on whatsapp", jid))
    }
    return ParseJID(jid)
}
```

**Modified function:**
```go
func ValidateJidWithLogin(client *whatsmeow.Client, jid string) (types.JID, error) {
    MustLogin(client)

    if config.WhatsappAccountValidation && !IsOnWhatsapp(client, jid) {
        return types.JID{}, pkgError.InvalidJID(fmt.Sprintf("Phone %s is not on whatsapp", jid))
    }

    phoneJID, err := ParseJID(jid)
    if err != nil {
        return types.JID{}, err
    }

    // Check if we have a LID mapping for this phone number
    // This ensures messages are sent to the correct contact when LID is known
    if client.Store != nil && client.Store.LIDs != nil {
        ctx := context.Background()
        lid, err := client.Store.LIDs.GetLIDForPN(ctx, phoneJID)
        if err == nil && !lid.IsEmpty() {
            logrus.Debugf("Using LID %s for phone number %s", lid.String(), phoneJID.String())
            return lid, nil
        }
    }

    return phoneJID, nil
}
```

### 2. Add import (if not present)
Add `"context"` to imports in `src/pkg/utils/whatsapp.go`

## Why This Works

1. **Centralized**: All 13 send functions call `ValidateJidWithLogin`, so the fix applies everywhere
2. **Backward Compatible**: If no LID mapping exists, falls back to phone JID
3. **Follows Existing Pattern**: Uses same nil-safety checks as `NormalizeJIDFromLID`
4. **No API Changes**: No changes to request structures or REST endpoints

## Affected Send Functions (Auto-Fixed)
All these functions call `ValidateJidWithLogin` and will automatically use LID when available:
- `SendText`, `SendImage`, `SendFile`, `SendVideo`, `SendAudio`
- `SendContact`, `SendLink`, `SendLocation`, `SendPoll`, `SendSticker`
- `SendChatPresence`, `getMentionFromText`, `getMentionsFromList`

## Verification

### 1. Build the project
```bash
cd /Users/milesibastos/code/chatwoot/go-whatsapp-web-multidevice/src
go build ./...
```

### 2. Run tests (if any)
```bash
go test ./...
```

### 3. Manual testing
1. Start go-whatsapp with debug logging enabled
2. Send a message to contact `5532988659372`
3. Check logs for: `Using LID X for phone number Y`
4. Verify message arrives at correct contact on WhatsApp device

### 4. Test cases
- Contact with known LID (received message before) - should use LID
- New contact without LID mapping - should use phone JID (fallback)
- Group messages - should work as before
