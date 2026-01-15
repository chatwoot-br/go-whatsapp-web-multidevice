# Plan: Sync Messages from Other WhatsApp Web Linked Devices

## Problem Statement
Messages sent from another WhatsApp Web linked device (e.g., device 23 at `174852883411040:23@lid`) arrive as `<unavailable/>` notifications. The whatsmeow library automatically requests the content via `BuildUnavailableMessageRequest()`, but the phone only acknowledges - it never returns the actual message content.

Additionally, history sync (which DOES contain these messages) only happens on initial QR code pairing, NOT on reconnects. After reconnect, the server returns `<offline count="0"/>` - no offline messages.

## Root Cause Analysis
```
13:43:01.353 <message from="174852883411040:23@lid" id="3EB0..."><unavailable/></message>
13:43:01.376 [SEND] Peer request to phone
13:43:01.701 <ack class="message" ...>  ← Phone acknowledges but never returns content
```

---

## Research Findings

### WhatsApp Protocol Limitations (from Official Docs)
- "Recent messages are securely synced to your companion device. Older chats stay on your primary phone"
- "You might not be able to view message history that hasn't been downloaded yet on the linked device"
- This is a **fundamental WhatsApp limitation**, not a bug in whatsmeow

### whatsmeow GitHub Issues
| Issue | Status | Finding |
|-------|--------|---------|
| [#654 - Requesting historical messages fail](https://github.com/tulir/whatsmeow/issues/654) | Closed (not planned) | `BuildHistorySyncRequest()` returns no response |
| [#195 - Message unavailable](https://github.com/tulir/whatsmeow/issues/195) | Fixed | Added auto-retry via `immediateRequestMessageFromPhone()` - but doesn't work for linked device messages |

### mautrix-whatsapp Implementation Analysis

**How they configure history sync** (from `connector.go`):
```go
store.DeviceProps.RequireFullSync = proto.Bool(wa.Config.HistorySync.RequestFullSync)
```

**Critical discovery**: `DeviceProps` (including `RequireFullSync`) is only sent during **registration** (`getRegistrationPayload()`), NOT during login/reconnection (`getLoginPayload()`).

This means:
- `RequireFullSync` only affects **initial QR code pairing**
- **Cannot request new full sync after pairing** without re-pairing
- mautrix-whatsapp has the **same limitation** we do

**How mautrix handles unavailable messages**:
- Creates placeholder/notice in chat
- Relies on retry mechanism (which doesn't work for linked device messages)
- **No special fix** for messages from other linked devices

### Unexplored Option: OnDemandReady Flags

In `store/clientpayload.go`, there are unexplored flags:
```go
HistorySyncConfig: &waCompanionReg.DeviceProps_HistorySyncConfig{
    OnDemandReady:         nil,  // Currently nil
    CompleteOnDemandReady: nil,  // Currently nil
    // ...
}
```

These might enable server-side ON_DEMAND history sync responses. Worth testing.

---

## Solution: Multi-Phase Approach

Given the protocol limitations, we propose a phased approach with realistic expectations.

### Phase 1: Enable Maximum Initial History Sync
**File**: `src/infrastructure/whatsapp/device_manager.go` or initialization code

Set `RequireFullSync=true` and enable OnDemand flags **before** device pairing:
```go
import "go.mau.fi/whatsmeow/store"

func init() {
    // Request full history on initial pairing (up to 1 year vs default 3 months)
    store.DeviceProps.RequireFullSync = proto.Bool(true)

    // Enable ON_DEMAND capability (experimental)
    store.DeviceProps.HistorySyncConfig.OnDemandReady = proto.Bool(true)
    store.DeviceProps.HistorySyncConfig.CompleteOnDemandReady = proto.Bool(true)
}
```

**Impact**: Only affects NEW device pairings. Existing devices need re-pairing.

### Phase 2: Handle UndecryptableMessage Events
**File**: `src/infrastructure/whatsapp/event_handler.go`

Add case for `events.UndecryptableMessage`:
```go
case *events.UndecryptableMessage:
    if evt.IsUnavailable {
        handleUnavailableMessage(ctx, evt, chatStorageRepo, client)
    }
```

### Phase 3: Request ON_DEMAND History Sync (Experimental)
**File**: `src/infrastructure/whatsapp/event_handler.go`

```go
var (
    unavailableChats    = make(map[string]time.Time)
    unavailableChatsMu  sync.RWMutex
    historySyncCooldown = 30 * time.Second
)

func handleUnavailableMessage(ctx context.Context, evt *events.UndecryptableMessage, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) {
    chatJID := evt.Info.Chat.String()

    // Check cooldown to prevent spamming
    unavailableChatsMu.Lock()
    lastRequest, exists := unavailableChats[chatJID]
    if exists && time.Since(lastRequest) < historySyncCooldown {
        unavailableChatsMu.Unlock()
        log.Debugf("[HISTORY_SYNC] Skipping request for %s (cooldown)", chatJID)
        return
    }
    unavailableChats[chatJID] = time.Now()
    unavailableChatsMu.Unlock()

    log.Infof("[HISTORY_SYNC] Requesting history for chat %s due to unavailable message %s", chatJID, evt.Info.ID)

    go requestChatHistory(ctx, client, evt.Info)
}

func requestChatHistory(ctx context.Context, client *whatsmeow.Client, msgInfo types.MessageInfo) {
    if client == nil || client.Store == nil || client.Store.ID == nil {
        return
    }

    msg := client.BuildHistorySyncRequest(&msgInfo, 50)
    _, err := client.SendMessage(ctx, client.Store.ID.ToNonAD(), msg, whatsmeow.SendRequestExtra{Peer: true})
    if err != nil {
        log.Errorf("[HISTORY_SYNC] Failed to request history for chat %s: %v", msgInfo.Chat.String(), err)
    } else {
        log.Infof("[HISTORY_SYNC] Sent ON_DEMAND request for chat %s", msgInfo.Chat.String())
    }
}
```

### Phase 4: Process ON_DEMAND Responses
**File**: `src/infrastructure/whatsapp/history_sync.go`

Update `processHistorySync()` to handle `ON_DEMAND` type:
```go
switch syncType {
case waHistorySync.HistorySync_INITIAL_BOOTSTRAP, waHistorySync.HistorySync_RECENT:
    return processConversationMessages(ctx, data, chatStorageRepo, client)
case waHistorySync.HistorySync_ON_DEMAND:
    return processOnDemandHistorySync(ctx, data, chatStorageRepo, client)
case waHistorySync.HistorySync_PUSH_NAME:
    return processPushNames(ctx, data, chatStorageRepo, client)
default:
    log.Debugf("Skipping history sync type: %s", syncType.String())
    return nil
}
```

Add new function to process and forward ON_DEMAND messages:
```go
func processOnDemandHistorySync(ctx context.Context, data *waHistorySync.HistorySync, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) error {
    log.Infof("Processing ON_DEMAND history sync with %d conversations", len(data.GetConversations()))

    // Store messages in database (reuse existing logic)
    if err := processConversationMessages(ctx, data, chatStorageRepo, client); err != nil {
        return err
    }

    // Forward individual messages to webhook (only for ON_DEMAND)
    if len(config.WhatsappWebhook) > 0 {
        for _, conv := range data.GetConversations() {
            for _, histMsg := range conv.GetMessages() {
                if histMsg == nil || histMsg.Message == nil {
                    continue
                }
                forwardHistorySyncMessageToWebhook(ctx, histMsg.Message, client)
            }
        }
    }
    return nil
}

func forwardHistorySyncMessageToWebhook(ctx context.Context, msg *waHistorySync.WebMessageInfo, client *whatsmeow.Client) {
    msgKey := msg.GetKey()
    if msgKey == nil {
        return
    }

    deviceID := ""
    if client != nil && client.Store != nil && client.Store.ID != nil {
        deviceJID := NormalizeJIDFromLID(ctx, client.Store.ID.ToNonAD(), client)
        deviceID = deviceJID.ToNonAD().String()
    }

    payload := map[string]any{
        "event":     "message",
        "device_id": deviceID,
        "payload": map[string]any{
            "id":                msgKey.GetID(),
            "from":              msgKey.GetRemoteJID(),
            "body":              utils.ExtractMessageTextFromProto(msg.GetMessage()),
            "timestamp":         time.Unix(int64(msg.GetMessageTimestamp()), 0).Format(time.RFC3339),
            "is_from_me":        msgKey.GetFromMe(),
            "from_history_sync": true,
        },
    }

    if err := forwardPayloadToConfiguredWebhooks(ctx, payload, "on_demand_message"); err != nil {
        log.Errorf("Failed to forward ON_DEMAND message to webhook: %v", err)
    }
}
```

---

## Files to Modify

| File | Phase | Change |
|------|-------|--------|
| `src/infrastructure/whatsapp/device_manager.go` | 1 | Set `RequireFullSync=true` and `OnDemandReady=true` |
| `src/infrastructure/whatsapp/event_handler.go` | 2-3 | Add `events.UndecryptableMessage` handler |
| `src/infrastructure/whatsapp/history_sync.go` | 4 | Handle `ON_DEMAND` sync type, forward messages to webhook |

---

## Verification Plan

### Phase 1 Testing (New Pairing)
```bash
cd src && go build -o ../go-whatsapp && cd .. && ./go-whatsapp
```
1. Delete existing device database
2. Pair new device via QR code
3. Verify logs show extended history sync:
   ```
   [INFO] Processing history sync type: INITIAL_BOOTSTRAP
   [INFO] Processing X conversations from history sync
   ```

### Phase 2-4 Testing (ON_DEMAND)
1. With paired device, send message from another WhatsApp Web session
2. Observe logs for:
   ```
   [WARN] Unavailable message XXX from YYY
   [INFO] Requesting history for chat ZZZ due to unavailable message XXX
   [INFO] Sent ON_DEMAND request for chat ZZZ
   [INFO] Processing ON_DEMAND history sync  ← This may NOT appear (known limitation)
   ```
3. If ON_DEMAND response received, verify webhook has `from_history_sync: true`

---

## Expected Outcomes & Risks

| Outcome | Likelihood | Notes |
|---------|------------|-------|
| Phase 1 works (more history on pairing) | **High** | Well-documented feature |
| ON_DEMAND requests get responses | **Low** | Issue #654 suggests this doesn't work |
| OnDemandReady flags help | **Unknown** | Never tested, worth trying |

### Fallback Options
1. **Document the limitation** - Messages from other WhatsApp Web sessions won't sync
2. **Re-pairing workflow** - Add UI option to force re-pairing to refresh history
3. **Periodic history sync** - On reconnect, trigger history sync for active chats (may hit rate limits)

---

## References
- [whatsmeow Issue #654 - Requesting historical messages fail](https://github.com/tulir/whatsmeow/issues/654)
- [whatsmeow Issue #195 - Message unavailable](https://github.com/tulir/whatsmeow/issues/195)
- [mautrix-whatsapp backfill.go](https://github.com/mautrix/whatsapp/blob/main/pkg/connector/backfill.go)
- [WhatsApp Help - Message history on linked devices](https://faq.whatsapp.com/653480766448040)
