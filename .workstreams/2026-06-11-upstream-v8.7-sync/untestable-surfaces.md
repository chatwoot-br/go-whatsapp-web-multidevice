# Untestable surfaces — paired-phone validation checklist

`whatsmeow.Client` is a concrete type (not an interface), so real-client call paths
can't be unit-mocked. These require human-owned paired-phone validation on a deploy
before merging to `origin/main`. (Filled during Phase A/B; mirrors the v8.5
`phase7-untestable.md`.)

## Real-client paths (nil-client branch covered; real branch needs a paired device)

- [ ] `ValidateAndNormalizeJID` → `client.IsOnWhatsApp` USync probe + BR 9th-digit-strip override (`pkg/utils/phone_br.go`)
- [ ] `NormalizeJIDFromLIDWithContext` → `client.Store.LIDs.GetPNForLID` (`jid_utils.go`, `event_message.go`, `history_sync.go`)
- [ ] `client.Store.Contacts.GetContact` contact-name lookup in webhook payload build (`event_message.go`)
- [ ] `deduplicateLIDChats` real LID resolution + orchestration (`history_sync.go`)
- [ ] `forwardHistorySyncCompleteToWebhook` device-ID from `client.Store.ID` (`forward_history_sync.go`)
- [ ] `client.SetProxyAddress` / `whatsmeow.SetProxyOptions` proxy wiring (`init.go`, `device_manager.go`)
- [ ] `client.MarkRead` read receipts (`event_message_handler.go`)

## Chatwoot end-to-end (needs a real Chatwoot instance)

- [ ] Standard webhook → chatwoot-app `whatsapp_web_controller` round-trip; HMAC `X-Hub-Signature-256` accepted
- [ ] Agent reply from Chatwoot UI → no double-post (fork `source_id` dedup)
- [ ] Message from physical phone → single mirrored outgoing Chatwoot message
- [ ] (Phase B) `session_id` field present + benign in chatwoot-app parser
