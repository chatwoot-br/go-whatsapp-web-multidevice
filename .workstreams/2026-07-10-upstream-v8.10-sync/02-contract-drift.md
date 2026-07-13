# Contract-drift check — v8.10.0 merge vs chatwoot-app webhook contract

Verdict: **0 breaking / 2 behavioral (benign) / HMAC stable.** The global-webhook path that
chatwoot-app's `Webhooks::WhatsappWebController` consumes is unchanged for fleet deployments
(no per-device webhook rows in the DB).

## What #671/#754/e4c62f8 changed in the delivery path

`forwardPayloadToConfiguredWebhooks` now resolves a per-device `DeviceWebhookConfig` from the
`devices` table (`GetDeviceRecordByJID` on payload `device_id`) before delivering:

- **URL selection**: device `webhook_url` if set, else `config.WhatsappWebhook` (global) — fleet
  instances have no device rows with `webhook_url` set → global path, unchanged.
- **Event whitelist**: `isEventWhitelistedForDevice(event, nil)` ≡ old
  `len(config.WhatsappWebhookEvents)==0 || isEventWhitelisted(event)` — verified in code.
- **HMAC**: `submitWebhook` falls back to `config.WhatsappWebhookSecret` when device config is
  nil/empty; `X-Hub-Signature-256: sha256=<hmac>` over raw JSON body unchanged (`webhook.go:56-63`).
- **Lookup failure**: falls back to global config with a warning — delivery never dropped on a
  config-read error (23620dc).

## Contract fields — verified intact

- Top-level `{event, device_id, payload}` + fork's `session_id` enrichment (`addWebhookSessionID`) ✓
- Fork-custom `history_sync_complete` event: `forward_history_sync.go` untouched by upstream;
  payload-shape test (`forward_history_sync_test.go`) green after stub-signature update ✓
- `chat_name`/`sender_name`: fork's 4-arg `forwardMessageToWebhook(ctx, client, evt, repo)` kept at
  the (restructured) `handleWebhookForward` call site; `buildEventPayload` chat_name lookup intact ✓
- Chatwoot routing (`forwardToChatwoot` + `shouldForwardEventToChatwoot`): unchanged; `message.ack`
  still gated by `ChatwootMessageRead` downstream (the call-site pre-gate upstream removed was
  redundant with this) ✓
- Receipt `Device == 0` duplicate guard: intact (`event_receipt.go:82-93`) ✓

## Behavioral changes (benign for the fleet)

1. **Broadcast skip hardened (e4c62f8)**: broadcast/status messages now early-return in
   `handleWebhookForward` regardless of Chatwoot — the fork's old gate already excluded broadcasts
   from the whole forward path, so no observable change for chatwoot-app consumers.
2. **Call-site config pre-gates removed** (`handleDeleteForMe`, `handleGroupInfo`,
   `handleAppState`, receipts): events are now always handed to
   `forwardPayloadToConfiguredWebhooks`, which no-ops when neither webhooks nor Chatwoot apply.
   Same deliveries, decision moved downstream (required for per-device webhooks to work when the
   global list is empty).

## New surface (additive, no consumer impact until used)

- `devices` table migrations 31–34 (webhook_url/secret/events/insecure_skip_verify), appended after 30.
- Device REST API can now set per-device webhook config; if a fleet instance ever sets one, that
  device's events divert to the device URL/secret — worth remembering operationally, not a drift today.
