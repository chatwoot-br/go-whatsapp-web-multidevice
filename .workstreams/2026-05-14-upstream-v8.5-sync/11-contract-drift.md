# 11 — Webhook contract-drift check (post-v8.5.0 standard path ↔ chatwoot-app fork controller)

Date: 2026-05-15. Read-only investigation. Scope per `10-q2-reversal.md`:
the one remaining "chatwoot integration ready" gate — does post-v8.5.0 GoWA's
**standard** webhook output (`forwardToWebhooks`, NOT `forwardToChatwoot`)
still match what chatwoot-app's fork-owned WhatsApp webhook controller parses.

GoWA repo: `go-whatsapp-web-multidevice` `main` `28f65b6`.
chatwoot-app: `chatwoot` `main` `287be988a`.

Conventions: `gowa:` = `/Users/woot/code/go-whatsapp-web-multidevice`,
`cw:` = `/Users/woot/code/chatwoot`. Facts cite `repo:path:line`. Inferences
are marked **[inference]**.

---

## Side 1 — chatwoot-app expects

### Endpoint + transport

- Route: `POST /webhooks/whatsapp_web` → `Webhooks::WhatsappWebController#process_payload`
  (`cw:fork/config/routes.rb:37`). No HMAC token in the path; `device_id`
  comes from the parsed body.
- Controller enqueues `Webhooks::WhatsappWebEventsJob.perform_later(params.to_unsafe_hash)`
  (`cw:fork/app/controllers/webhooks/whatsapp_web_controller.rb:30`); the job
  hands off to `Whatsapp::IncomingMessageWhatsappWebService`
  (`cw:fork/app/jobs/webhooks/whatsapp_web_events_job.rb:21`).

### HMAC verification

- `verify_webhook_secret` (`cw:fork/app/controllers/webhooks/whatsapp_web_controller.rb:55-68`):
  reads header `X-Hub-Signature-256`, computes
  `OpenSSL::HMAC.hexdigest(sha256, secret, request.raw_post)`, compares
  `"sha256=#{hex}"` with `ActiveSupport::SecurityUtils.secure_compare`.
- Secret source: `ENV['WHATSAPP_WEB_WEBHOOK_SECRET']`
  (`cw:...whatsapp_web_controller.rb:56`). Blank secret → verification
  skipped (fail-open dev ergonomics, line 57). What is signed: the **raw
  POST body** (`request.raw_post`).

### Top-level keys read

`device_id`, `event`, `payload`
(`cw:fork/app/services/whatsapp/incoming_message_whatsapp_web_service.rb:43-51`;
controller reads `params[:device_id]` at `whatsapp_web_controller.rb:8`).

### Event field + values branched on

`transform_webhook_payload` switches on `webhook_params[:event]`
(`cw:...incoming_message_whatsapp_web_service.rb:51-70`):

| `event` value | Handling |
|---|---|
| `message` | `transform_message_event` |
| `message.ack` | `transform_status_event` |
| `message.reaction` | `transform_reaction_event` |
| `history_sync_complete` | `process_history_sync` (fork-custom) |
| `message.revoked`, `message.edited` | explicit no-op `{}` |
| anything else | `Rails.logger.warn "Unknown WhatsApp Web event type: #{event_type}"`, returns `{}` — **graceful-ignore, no hard-fail** (`:68-69`) |

### Nested `payload` fields read

- Message: `chat_id`, `id`, `timestamp`, `from`, `from_name`, `is_from_me`,
  `from_lid`, `body`, `replied_to_id`, `caption`; media keys `image`,
  `video`, `video_note`, `audio`, `document`, `sticker`, `location`,
  `live_location`, `contact` (`cw:...:73-97, 537-607`).
- Fork-custom contact-name fields: `chat_name`, `contact_name`
  (`cw:...:308, 546-548`), `from_name` (`:444, 481, 551`).
- Status (`message.ack`): `ids` (array), `receipt_type`
  (`cw:...:107-126`). Maps `delivered`→delivered; `read`/`read_self`/
  `played`/`played_self`→read; anything else → `{}` ignored.
- Reaction: `reacted_message_id`, `reaction`, `from` (`cw:...:130-138`).
- Media object: reads `media_path` OR `url`, plus nested `caption`
  (`cw:...:609-625`).

### Optional-field / echo handling

- Unknown event → graceful-ignore (above).
- Outgoing/echo (`is_from_me == true`): NOT rejected. `transform_message_event`
  treats `chat_id` as the contact for `is_from_me` (`cw:...:87-91`);
  `message_attributes` builds an **outgoing** message with a device-contact
  sender (`cw:...:414-419, 434-461`).
- Missing optional fields: `.dig`/`[]` with `.present?`/`.blank?` guards
  throughout; no hard-fail on absent keys.
- **Dedup**: `create_message` looks up `inbox.messages.find_by(source_id:)`
  before building; if found it updates-or-returns the existing message and
  sets `@message_already_exists` (`cw:...:387-401`).

### Fork-custom field dependence

- `chat_name`: read in `set_lid_contact` (`cw:...:308`) and `build_contact`
  (`cw:...:548`) for outgoing-message contact naming.
- `sender_name`: read as `message['sender_name']` in
  `lookup_sender_name_for_group` (`cw:...:673`) for group-message sender
  resolution.
- `history_sync_complete`: dedicated event branch → `process_history_sync`
  (`cw:...:61-63, 842`).

---

## Side 2 — GoWA v8.5.0 emits (standard path)

### Transport + HMAC

- `submitWebhook` marshals `payload` to JSON, signs with
  `utils.GetMessageDigestOrSignature(postBody, []byte(config.WhatsappWebhookSecret))`,
  sets header `X-Hub-Signature-256: sha256=<hex>`
  (`gowa:src/infrastructure/whatsapp/webhook.go:31-48`).
- `GetMessageDigestOrSignature` is **HMAC-SHA256, hex-encoded**
  (`gowa:src/pkg/utils/whatsapp.go:649-656`: `hmac.New(sha256.New, key)`
  → `hex.EncodeToString`).
- Secret default = literal `"secret"` (`gowa:src/config/settings.go:33`,
  `WhatsappWebhookSecret = "secret"`), env `WHATSAPP_WEBHOOK_SECRET`.
- Signed input = the marshalled POST body (`webhook.go:31-42`).

### Top-level structure

`{ "event": <string>, "device_id": <non-AD JID>, "payload": {...} }`
— `WebhookEvent` (`gowa:src/infrastructure/whatsapp/event_message.go:32-37`,
`:52-56`).

### Event taxonomy emitted on the standard path

| `event` | Source |
|---|---|
| `message` | `event_message.go:26, 187` |
| `message.reaction` | `event_message.go:27, 174` |
| `message.revoked` | `event_message.go:28, 151` |
| `message.edited` | `event_message.go:29, 164` |
| `message.ack` | `event_receipt.go:71` |
| `chat_presence` | `event_chat_presence.go:67` |
| `call.offer` | `event_call.go:81` |
| `history_sync_complete` | `forward_history_sync.go:31` (fork-only) |
| `on_demand_message` | `history_sync.go:386` (wraps `event: "message"`) |

### Message payload fields emitted

- Common: `id`, `timestamp` (**RFC3339**, `event_message.go:92`),
  `is_from_me` (**always present**, `:93`).
- `chat_id`, `from`; `from_lid` only when sender is LID; `chat_lid` when
  chat is LID (`event_message.go:190-205`).
- `from_name` = pushname when non-empty (`:99-101`).
- `chat_name` = lookup chain (chat storage → contact store → contact
  pushname) on `is_from_me` (`:105-136`) — **fork delta, recovered in
  `25f494b`**.
- `body`, `replied_to_id`, `quoted_body` (`:231-249`).
- Media: `image`/`video`/`video_note` → `{url,caption}` or auto-download
  `{path,caption}` or bare path; `audio`/`sticker` → `{url}` or path;
  `document` → `{url,filename}` (`event_message.go:277-385`).
- Optional: `view_once`, `forwarded`, `referral`, `contact`,
  `contacts_array`, `list`, `live_location`, `location`, `order`
  (`event_message.go:255-411`).
- `sender_name`: present on the **chat domain** json tag
  (`src/domains/chat/chat.go`, restored in `25f494b`); read by chatwoot-app
  history-sync path as `message['sender_name']`.

### `message.ack` payload

`ids` (array), `chat_id`, `from`, `from_lid?`, `receipt_type`,
`receipt_type_description` (new), `timestamp`
(`gowa:src/infrastructure/whatsapp/event_receipt.go:43-78`). Only
primary-device receipts forwarded (`event_receipt.go:88-94`).

### `history_sync_complete` payload

`{ "event":"history_sync_complete", "device_id":<JID>,
"payload":{ "sync_type":..., "timestamp":<RFC3339> } }`
(`gowa:src/infrastructure/whatsapp/forward_history_sync.go:30-37`).

### Outgoing-message forwarding (the load-bearing change)

- **Pre-upgrade** (`909b6e6`, parent of the v8.5 sync), the fork's
  `event_message_handler.go` had an explicit gate:
  `if evt.Info.IsFromMe && !config.WhatsappWebhookIncludeOutgoing { skip }`
  (confirmed via `git show 3b87f4e~1:src/infrastructure/whatsapp/event_message_handler.go`,
  lines 117-121).
- The v8.5 upstream-sync commit `3b87f4e` **deleted those 9 lines**
  (`git show 3b87f4e --stat`: `event_message_handler.go | 9 -`).
- **Post-upgrade** (`28f65b6`) `handleWebhookForward`
  (`gowa:src/infrastructure/whatsapp/event_message_handler.go:102-127`)
  has **no `IsFromMe` / INCLUDE_OUTGOING gate**. Outgoing messages are
  forwarded unconditionally (only broadcast + non-REVOKE/EDIT protocol
  messages are skipped).
- `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` survives only as a dead line in
  `gowa:src/.env.example:24`. No Go code reads it (grep of `src/config/`,
  `src/cmd/`, `src/infrastructure/` returns nothing). **Dead env var.**

---

## Delta analysis

| # | Item | Class | Evidence | Required fix |
|---|---|---|---|---|
| 1 | HMAC scheme/header/secret | **CONFIRMED-STABLE** | GoWA: HMAC-SHA256 hex, `X-Hub-Signature-256: sha256=<hex>`, secret `WHATSAPP_WEBHOOK_SECRET` default `"secret"` (`gowa:webhook.go:41-48`, `gowa:whatsapp.go:649-656`). chatwoot-app verifies identically over `request.raw_post` (`cw:whatsapp_web_controller.rb:55-68`). Both ends share the secret value out-of-band. | none |
| 2 | Top-level keys (`event`/`device_id`/`payload`) | **CONFIRMED-STABLE** | GoWA emits exactly these (`gowa:event_message.go:52-56`); chatwoot-app reads exactly these (`cw:...:43-51`). | none |
| 3 | Core message fields (`id`,`chat_id`,`from`,`from_name`,`from_lid`,`body`,`replied_to_id`,media) | **CONFIRMED-STABLE** | Field-by-field match between `gowa:event_message.go:91-411` and `cw:...:537-625`. Media object dual-shape (`media_path`/`url` + nested `caption`) handled (`cw:...:609-625`). | none |
| 4 | Fork deltas `chat_name`/`sender_name`/`history_sync_complete` | **CONFIRMED-STABLE** | `chat_name` emitted (`gowa:event_message.go:131-132`), read (`cw:...:308,548`). `sender_name` restored on chat domain in `25f494b`, read (`cw:...:673`). `history_sync_complete` event emitted (`gowa:forward_history_sync.go:31`), branched (`cw:...:61`). | none |
| 5 | `timestamp` format: Unix-int-string → **RFC3339** | **CONFIRMED-STABLE** (silent shift, absorbed) | GoWA emits `time.RFC3339` (`gowa:event_message.go:92`). chatwoot-app `parse_timestamp` has an `else Time.zone.parse(timestamp_str)` branch for non-numeric strings (`cw:...:727-738`); `message_created_at` falls back to `Time.current` on `ArgumentError` (`cw:...:423-430`). Parser absorbs both. **[inference]** old payload was Unix-int (typical GoWA); the RFC3339 branch was already present in the fork parser, so this is non-breaking. | none (note in Phase-10 verification) |
| 6 | `message.ack` adds `receipt_type_description` | **ADDITIVE (safe)** | New key (`gowa:event_receipt.go:68`). chatwoot-app `transform_status_event` only reads `ids`+`receipt_type` (`cw:...:107-126`); extra key ignored. | none |
| 7 | New events `chat_presence`, `call.offer` | **ADDITIVE (safe, log-volume risk)** | Emitted (`gowa:event_chat_presence.go:67`, `event_call.go:81`). Fall through chatwoot-app's `else → Rails.logger.warn "Unknown WhatsApp Web event type"` (`cw:...:67-69`). `chat_presence` (typing/online) can fire very frequently → **log-noise + redundant Sidekiq job churn**. | optional: gate `chat_presence`/`call.offer` out via `WHATSAPP_WEBHOOK_EVENTS` whitelist on the GoWA deploy (config, not code) |
| 8 | New optional msg fields `contacts_array`,`view_once`,`forwarded`,`referral`,`quoted_body`,`list`,`order`,`chat_lid` | **ADDITIVE (safe)** | Emitted (`gowa:event_message.go:255-411`). chatwoot-app `determine_message_content` only branches on a fixed key set (`cw:...:586-602`); unrecognized keys ignored, message degrades to its text body. | none |
| 9 | `is_from_me` always present + **outgoing always forwarded** (INCLUDE_OUTGOING gate deleted by `3b87f4e`) | **BEHAVIORAL (needs handling — analyzed below)** | Pre: gate at `event_message_handler.go:121` (`3b87f4e~1`). Post: no gate (`gowa:event_message_handler.go:102-127`). | see Behavioral analysis; **no chatwoot-app code change required** (dedup already covers the UI-reply echo) — verification + optional config only |

### Behavioral item #9 — outgoing-echo analysis (highest suspicion)

Two distinct echo sub-cases once the gate is gone:

**(a) Agent reply sent from the Chatwoot UI → echoed back as `is_from_me=true`.**
Chatwoot's outbound flow persists GoWA's returned message id as the
message's `source_id`:
`Whatsapp::Providers::WhatsappWebService#process_response` returns
`results.message_id` (`cw:fork/app/services/whatsapp/providers/whatsapp_web_service.rb:189-197`);
`Whatsapp::SendOnWhatsappService` does
`message.update!(source_id: message_id)`
(`cw:app/services/whatsapp/send_on_whatsapp_service.rb:42`). When the echo
webhook arrives with the same WhatsApp message id in `payload[:id]`
(**[inference]** — the `results.message_id` GoWA returns from `/send/message`
is the same WhatsApp message id echoed back in the `is_from_me` webhook;
this is the whatsmeow norm but is not traced end-to-end here; UAT item 1
verifies it empirically), `create_message` finds the existing message via
`inbox.messages.find_by(source_id:)` and short-circuits
(`cw:fork/app/services/whatsapp/incoming_message_whatsapp_web_service.rb:387-401`).
**Result: UI-sent agent replies do NOT double-post. The fork parser already
dedups them by `source_id`.** This is the highest-risk hypothesis and it
resolves **safe**.

**(b) Message typed by the operator on the *physical phone* → echoed as
`is_from_me=true`.** No prior Chatwoot record exists, so `source_id` dedup
does not fire; the parser creates a fresh **outgoing** message attributed to
a synthetic device contact (`cw:...:414-461`). This is **new, intended
behavior** (the phone-side half of the conversation now mirrors into
Chatwoot) rather than a defect — but it is a behavior change operators
should be told about (phone replies will appear in Chatwoot threads). Not a
double-post. No code fix; document + UAT-observe.

**Conclusion:** the deleted gate does not cause agent-reply double-posting;
the fork's `source_id` dedup, present independently of the gate, covers
case (a). Case (b) is additive conversation mirroring. **Zero required
chatwoot-app code change for #9.**

---

## Verdict

**Contract is clean. 0 BREAKING, 1 BEHAVIORAL (already safely absorbed by
existing fork dedup), 2 ADDITIVE-with-operational-notes.**

- No field chatwoot-app keys on was renamed/removed/retyped.
- No hard-branched event changed name.
- HMAC scheme/header/secret-source unchanged on the standard path
  (OQ1's "HMAC path untouched" verified specifically for `forwardToWebhooks`).
- The one behavioral shift (outgoing always forwarded) is neutralized by the
  fork's pre-existing `source_id` dedup; the only residual effect is benign
  conversation mirroring + log/job volume from `chat_presence`.

The Phase-10 "chatwoot integration ready" gate is **satisfied with no
mandatory code change**. Remaining work is verification + optional
deploy-config hardening.

---

## Scoped Phase-10 fix list

Ordered. No mandatory chatwoot-app code change. All items are
verification or deploy-config; effort in hours.

1. **[verify, ~1h]** Real-gateway UAT: send an agent reply from Chatwoot
   UI, confirm the `is_from_me=true` echo is deduped (no second message).
   Assertion point: `cw:fork/app/services/whatsapp/incoming_message_whatsapp_web_service.rb:389`
   (`find_by(source_id:)` hit). This also empirically confirms the
   message-id-identity inference in Behavioral #9a and, incidentally, that
   the RFC3339 `timestamp` (delta #5) yields a correct `created_at`. Reuse
   the Phase-5 real-gateway harness.
2. **[verify, ~0.5h]** UAT: send a message from the physical phone, confirm
   it appears as a single outgoing Chatwoot message via the device contact
   (`cw:...:434-461`) and is not duplicated on retry.
3. **[config, ~0.5h]** On the GoWA deploy, set `WHATSAPP_WEBHOOK_EVENTS` to
   the whitelist chatwoot-app consumes (`message,message.ack,message.reaction,history_sync_complete`)
   to suppress `chat_presence`/`call.offer` log-noise + Sidekiq churn
   (delta item #7). Pure env config (`gowa:src/config/settings.go:35`,
   `isEventWhitelisted` `gowa:webhook_forward.go:474-481`); no code.
4. **[doc, ~0.5h]** Note in operator runbook: phone-side replies now mirror
   into Chatwoot as outgoing messages (behavioral item #9b) — expected, not
   a bug. Also note that, until item 3's whitelist is applied, every echo
   (and every `chat_presence`) still runs `set_contact` +
   `find_or_create_device_contact` + avatar-sync side-effects (idempotent
   but Sidekiq-job-heavy) before the `source_id` dedup short-circuits the
   duplicate row — so item 3 is a throughput fix, not merely log hygiene.

**Total estimate: ~2.5 person-hours** (≈ half a day with setup/teardown).
No multi-day work. No port, no migration, no native cutover (per
`10-q2-reversal.md`).

### Rebase-surface confirmation

**Zero allowlist touches — confirmed.** Every file implicated above
(`cw:fork/app/services/whatsapp/incoming_message_whatsapp_web_service.rb`,
`cw:fork/app/services/whatsapp/providers/whatsapp_web_service.rb`,
`cw:fork/app/controllers/webhooks/whatsapp_web_controller.rb`) lives under
`fork/**`. The rebase-surface allowlist at
`cw:fork/bin/check-rebase-surface.sh:70-83` is an 11-entry OSS-tree list
with **no WhatsApp file**, and the check explicitly excludes `fork/**` via
the `:!fork/**` pathspec at `cw:fork/bin/check-rebase-surface.sh:60-61`.
Since Phase-10 requires no chatwoot-app code change anyway, the touched-file
count is zero regardless. Both repos remain read-only under this
investigation; the only file written is this report.
