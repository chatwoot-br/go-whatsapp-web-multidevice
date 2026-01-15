# ISSUE-001: Messages from Other Linked Devices Not Synced

## Status: Open (Protocol Limitation)

## Summary
Messages sent from another WhatsApp Web linked device (e.g., browser WhatsApp Web) do not arrive in go-whatsapp-web-multidevice, preventing them from being forwarded to Chatwoot webhooks.

## Environment
- **go-whatsapp-web-multidevice**: v4.9.1+2
- **whatsmeow**: v0.0.0-20251217143725-11cf47c62d32
- **Connected phone**: 552140402221
- **Other linked device**: WhatsApp Web (device 23 at `174852883411040:23@lid`)

## Problem Description

### Observed Behavior
When a message is sent from another WhatsApp Web session (not go-whatsapp), the following occurs:

1. go-whatsapp receives an `<unavailable/>` notification instead of the actual message
2. whatsmeow automatically requests the message content via `BuildUnavailableMessageRequest()`
3. The phone acknowledges the request but **never returns the actual message content**
4. The message is effectively lost and never forwarded to Chatwoot

### Log Evidence
```
13:43:01.353 <message from="174852883411040:23@lid" id="3EB0625526AA1D1CF5DA54" ...>
               <unavailable/>
             </message>
13:43:01.353 [WARN] Unavailable message 3EB0625526AA1D1CF5DA54
13:43:01.376 [SEND] Peer request to phone (BuildUnavailableMessageRequest)
13:43:01.701 <ack class="message" ...>  ← Phone acknowledges
13:43:10.145 <receipt type="peer_msg">  ← Phone sends receipt
❌ NO message content ever returned
```

### Expected Behavior
Messages sent from any linked device should be received and forwarded to configured webhooks.

## Root Cause Analysis

### WhatsApp Multi-Device Protocol Limitation
This is a **fundamental limitation of the WhatsApp multi-device protocol**, not a bug in go-whatsapp or whatsmeow.

From [WhatsApp Help Center](https://faq.whatsapp.com/653480766448040):
> "Recent messages are securely synced to your companion device. Older chats stay on your primary phone."
> "You might not be able to view message history that hasn't been downloaded yet on the linked device."

### Technical Details

1. **Unavailable Messages**: When another linked device sends a message, other companion devices receive an `<unavailable/>` notification instead of the encrypted message content.

2. **Retry Mechanism Doesn't Work**: whatsmeow's `immediateRequestMessageFromPhone()` sends a request for the message content, but the phone only acknowledges without returning the data.

3. **History Sync Only at Pairing**: Full message history is only synced during initial QR code pairing. Reconnections receive `<offline count="0"/>` - no offline messages.

4. **ON_DEMAND History Sync Unreliable**: `BuildHistorySyncRequest()` for on-demand history fetching doesn't receive responses ([whatsmeow Issue #654](https://github.com/tulir/whatsmeow/issues/654)).

## Research Findings

### whatsmeow GitHub Issues

| Issue | Status | Finding |
|-------|--------|---------|
| [#654](https://github.com/tulir/whatsmeow/issues/654) | Closed (not planned) | `BuildHistorySyncRequest()` returns no response |
| [#195](https://github.com/tulir/whatsmeow/issues/195) | Fixed | Added auto-retry - but doesn't work for linked device messages |

### mautrix-whatsapp Bridge
The production [mautrix-whatsapp](https://github.com/mautrix/whatsapp) bridge (which uses whatsmeow) has the **same limitation**. They:
- Create placeholder notices for unavailable messages
- Rely on retry mechanism (which doesn't work)
- Have no special fix for linked device messages

### Key Code References

**whatsmeow retry mechanism** (`retry.go`):
```go
func (cli *Client) immediateRequestMessageFromPhone(chat, sender types.JID, id types.MessageID) {
    msg := cli.BuildUnavailableMessageRequest(chat, sender, id)
    _, err := cli.SendMessage(context.Background(), cli.Store.ID.ToNonAD(), msg, SendRequestExtra{Peer: true})
    // Phone acknowledges but never returns content
}
```

**DeviceProps only sent at registration** (`store/clientpayload.go`):
```go
func (device *Device) getRegistrationPayload() *waWa6.ClientPayload {
    // DeviceProps (including RequireFullSync) sent here
    payload.DevicePairingData = &waWa6.ClientPayload_DevicePairingRegistrationData{
        DeviceProps: deviceProps,
    }
}

func (device *Device) getLoginPayload() *waWa6.ClientPayload {
    // DeviceProps NOT sent on reconnection
    // Cannot request new history sync after pairing
}
```

## Impact

### Affected Use Cases
1. **Customer support teams** using multiple WhatsApp Web sessions
2. **Business accounts** where agents respond from different devices
3. **Any scenario** where messages are sent from WhatsApp Web while go-whatsapp is the connected companion device

### Severity
**Medium-High**: Messages are silently lost, potentially causing missed customer inquiries.

## Potential Mitigations

### Implemented/Possible Solutions

| Solution | Feasibility | Notes |
|----------|-------------|-------|
| Enable `RequireFullSync=true` on pairing | ✅ Works | Only affects initial pairing, gets more history |
| Enable `OnDemandReady` flags | ❓ Untested | May enable ON_DEMAND responses |
| Request ON_DEMAND history on unavailable | ⚠️ Unreliable | Issue #654 suggests this doesn't work |
| Force re-pairing to refresh history | ✅ Works | Poor UX, loses current session |
| Document limitation | ✅ Feasible | Users should be aware |

### Recommended Approach
See [Plan: sync-messages-from-linked-devices.md](../plans/sync-messages-from-linked-devices.md) for detailed implementation plan.

## Workarounds for Users

1. **Use only go-whatsapp for sending**: Don't use other WhatsApp Web sessions for the connected number
2. **Re-pair periodically**: Disconnect and re-scan QR code to get fresh history sync
3. **Use phone app**: Messages sent from the phone app are properly synced

## Related Files
- `src/infrastructure/whatsapp/event_handler.go` - Event handling
- `src/infrastructure/whatsapp/history_sync.go` - History sync processing
- `go.mau.fi/whatsmeow/retry.go` - Message retry logic
- `go.mau.fi/whatsmeow/store/clientpayload.go` - Device registration

## References
- [whatsmeow Issue #654 - Requesting historical messages fail](https://github.com/tulir/whatsmeow/issues/654)
- [whatsmeow Issue #195 - Message unavailable](https://github.com/tulir/whatsmeow/issues/195)
- [mautrix-whatsapp backfill.go](https://github.com/mautrix/whatsapp/blob/main/pkg/connector/backfill.go)
- [WhatsApp Help - Message history on linked devices](https://faq.whatsapp.com/653480766448040)

## Timeline
- **2025-01-15**: Issue identified during Chatwoot integration debugging
- **2025-01-15**: Root cause analysis completed
- **2025-01-15**: Confirmed as WhatsApp protocol limitation (not fixable)
