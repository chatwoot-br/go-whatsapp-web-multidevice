# ISSUE-001: Messages from Other Linked Devices Not Synced

## Status: Open (Protocol Limitation - Mitigations Implemented)

## Summary
Messages sent from another WhatsApp Web linked device (e.g., browser WhatsApp Web) do not arrive in go-whatsapp-web-multidevice, preventing them from being forwarded to Chatwoot webhooks.

## Environment
- **go-whatsapp-web-multidevice**: v4.9.1+2
- **whatsmeow**: v0.0.0-20251217143725-11cf47c62d32
- **Test device (go-whatsapp)**: 5521995539939 (device 24)
- **Other linked device**: WhatsApp Web "ChatWoot (DEV)" (device 23 at `186234680901688:23@lid`)
- **Recipient phone**: 5521998762522

## Problem Description

### Observed Behavior
When a message is sent from another WhatsApp Web session (not go-whatsapp), the following occurs:

1. go-whatsapp receives an `<unavailable/>` notification instead of the actual message
2. whatsmeow automatically requests the message content via `BuildUnavailableMessageRequest()`
3. The phone acknowledges the request but **never returns the actual message content**
4. The message is effectively lost and never forwarded to Chatwoot

### Log Evidence (2026-01-15)

**Scenario**: Message sent from WhatsApp Web "ChatWoot (DEV)" (device 23) to contact 5521998762522

```
18:11:41.725 <message from="186234680901688:23@lid" id="3EB03C6D5393C95EA743D2"
             notify="ChatWoot (DEV)" recipient="151474956939293@lid" type="text">
               <unavailable/>
             </message>

18:11:41.725 [WARN] Unavailable message 3EB03C6D5393C95EA743D2 from 186234680901688:23@lid
             in 151474956939293@lid (type: "")

18:11:41.725 [WARN] [UNAVAILABLE_MSG] Message 3EB03C6D5393C95EA743D2 from 186234680901688:23@lid
             in chat 151474956939293@lid is unavailable (from another linked device)

18:11:41.725 [INFO] [HISTORY_SYNC] Requesting ON_DEMAND history for chat 151474956939293@lid
             due to unavailable message 3EB03C6D5393C95EA743D2

18:11:41.846 Requested message 3EB03C6D5393C95EA743D2 from phone

18:11:41.964 [INFO] [HISTORY_SYNC] Sent ON_DEMAND request for chat 151474956939293@lid
             (response may not arrive - protocol limitation)

18:11:42.358 <receipt from="151474956939293@lid" id="3EB03C6D5393C95EA743D2" type="read"/>
             ← Delivery/read receipts arrive, but message content NEVER arrives
```

**Key observation**: The message was marked as delivered and read (receipts arrived), but the actual content was **never received**.

### Reconnect Does NOT Fetch Missing Messages

When user clicks "Reconnect" from Chatwoot UI, history sync is NOT triggered:

```
18:18:55 | POST | /devices/5521995539939/reconnect   ← User clicked "Reconnect"
18:18:57.893 <ib from="s.whatsapp.net"><offline count="0"/></ib>  ← Server: NO offline messages!
```

| Scenario | History Sync? | What Server Returns |
|----------|---------------|---------------------|
| **Initial QR pairing** | ✅ Yes | RECENT, FULL, PUSH_NAME syncs |
| **Reconnect** | ❌ No | `<offline count="0"/>` |
| **WebSocket auto-reconnect** | ❌ No | `<offline count="0"/>` |

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

5. **DeviceProps Only at Registration**: `RequireFullSync` and `OnDemandReady` flags are only sent during device registration (QR pairing), not on reconnection.

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

## Implemented Mitigations

### Phase 1: Enable Maximum Initial History Sync ✅
**Commit**: `74afcdb` - `feat(whatsapp): enable full history sync and ON_DEMAND capability`

**File**: `src/infrastructure/whatsapp/device_manager.go`
```go
store.DeviceProps.RequireFullSync = proto.Bool(true)
store.DeviceProps.HistorySyncConfig.OnDemandReady = proto.Bool(true)
store.DeviceProps.HistorySyncConfig.CompleteOnDemandReady = proto.Bool(true)
```
**Impact**: Only affects NEW device pairings. Gets up to 1 year of history vs default 3 months.

### Phase 2-3: Handle Unavailable Messages ✅
**Commit**: `73be9eb` - `feat(whatsapp): handle unavailable messages from linked devices`

**File**: `src/infrastructure/whatsapp/event_handler.go`
- Added `events.UndecryptableMessage` handler
- Logs unavailable messages with `[UNAVAILABLE_MSG]` prefix
- Requests ON_DEMAND history sync with 30-second cooldown

### Phase 4: Process ON_DEMAND Responses ✅
**Commit**: `cebd4bf` - `feat(whatsapp): process ON_DEMAND history sync responses`

**File**: `src/infrastructure/whatsapp/history_sync.go`
- Added `ON_DEMAND` case to history sync processing
- Forwards ON_DEMAND messages to webhooks with `from_history_sync: true`

### Mitigation Results

| Mitigation | Status | Result |
|------------|--------|--------|
| `RequireFullSync=true` | ✅ Implemented | Works for new pairings |
| `OnDemandReady` flags | ✅ Implemented | No response received (as expected) |
| ON_DEMAND request on unavailable | ✅ Implemented | Request sent, no response received |
| Detect & log unavailable messages | ✅ Implemented | Working - see logs above |

**Conclusion**: All mitigations implemented, but ON_DEMAND responses do not arrive due to WhatsApp protocol limitations. This confirms the issue is **not fixable** without changes to WhatsApp's protocol.

## Workarounds for Users

1. **Use only go-whatsapp for sending**: Don't use other WhatsApp Web sessions for the connected number
2. **Re-pair periodically**: Disconnect and re-scan QR code to get fresh history sync
3. **Use phone app**: Messages sent from the phone app are properly synced to all devices
4. **Single WhatsApp Web session**: If using go-whatsapp as primary, avoid opening WhatsApp Web in browser

## Related Files
- `src/infrastructure/whatsapp/device_manager.go` - DeviceProps configuration
- `src/infrastructure/whatsapp/event_handler.go` - UndecryptableMessage handler
- `src/infrastructure/whatsapp/history_sync.go` - History sync processing, ON_DEMAND handling
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
- **2026-01-15**: Phase 1-4 mitigations implemented
- **2026-01-15**: Log investigation confirmed:
  - Unavailable messages correctly detected and logged
  - ON_DEMAND requests sent but no response received
  - Reconnect returns `<offline count="0"/>` - no history sync on reconnect
  - Protocol limitation confirmed - messages from other linked devices cannot be synced
