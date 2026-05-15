# chatwoot-app ↔ GoWA integration topology (v8.5 sync scoping)

Read-only investigation. GoWA repo `/Users/woot/code/go-whatsapp-web-multidevice` @ `main` (28f65b6).
chatwoot-app repo `/Users/woot/code/chatwoot` @ `main` (287be988a).

Citations: `gowa:` = go-whatsapp-web-multidevice repo; `cw:` = chatwoot fork repo.

## A. GoWA side

The upstream-native chatwoot integration is **API-direct push** for inbound (WhatsApp→Chatwoot) and **Chatwoot-standard-webhook receive** for outbound (Chatwoot→WhatsApp). It is NOT an outbound custom webhook to a non-Chatwoot URL.

**Inbound — GoWA pushes to Chatwoot's public REST API.** Auth is the `api_access_token` header carrying `config.ChatwootAPIToken`; account/inbox IDs come from config (`gowa:src/infrastructure/chatwoot/client.go:92-106`). Exact endpoints called:

- Contact search: `GET {BaseURL}/api/v1/accounts/{AccountID}/contacts/search?q=...` (`gowa:src/infrastructure/chatwoot/client.go:109-125`)
- Create contact: `POST .../accounts/{AccountID}/contacts` (`gowa:client.go:166`); stores WhatsApp JID as custom attribute `waha_whatsapp_jid` (`gowa:client.go:183-186`)
- Update contact name: `PUT .../accounts/{AccountID}/contacts/{id}` (`gowa:client.go:264`)
- Contact conversations: `GET .../accounts/{AccountID}/contacts/{id}/conversations` (`gowa:client.go:298`)
- Create conversation: `POST .../accounts/{AccountID}/conversations` (`gowa:client.go:345`)
- Create message: `POST .../accounts/{AccountID}/conversations/{id}/messages` (JSON, or multipart for attachments) (`gowa:client.go:406-409,452`)

WhatsApp events reach this path via `forwardToChatwoot` — a goroutine off the webhook-forward fan-out, gated on `config.ChatwootEnabled` and event `message`/`message.reaction` (`gowa:src/infrastructure/whatsapp/webhook_forward.go:74-95,223-230,433-471`). It calls `FindOrCreateContact` → `FindOrCreateConversation` → `CreateMessage` (`gowa:webhook_forward.go:399-431`). Reactions post a NEW message ("X reacted 👍 to message …"), not an update to the original (`gowa:webhook_forward.go:242-264,460-461`).

**Outbound — `/chatwoot/webhook` receives Chatwoot→GoWA agent replies.** Registered before basic-auth middleware so Chatwoot can post unauthenticated (`gowa:src/cmd/rest.go:92-100`; auth-exclusion + history-sync added in commit `3b87f4e` "Chatwoot message history sync & webhook auth fix"). It parses Chatwoot's **standard webhook payload** (`event`, `message_type`, `conversation.meta.sender`, `attachments`) (`gowa:src/infrastructure/chatwoot/types.go:47-78`). It acts only on `event=message_created` + `message_type=outgoing` + `private=false`, then sends to WhatsApp via the destination resolved from the contact's `waha_whatsapp_jid` custom attr or phone (`gowa:src/ui/rest/chatwoot.go:66-145`). Echo-loop guard: messages GoWA created via the API are registered in `sentMessageIDs` and skipped when they bounce back as webhooks (`gowa:client.go:31-90`, `gowa:chatwoot.go:78-81`).

**`/chatwoot/sync*` — history import, GoWA→Chatwoot, API-direct.** `POST /chatwoot/sync` (auth-required, `gowa:src/cmd/rest.go:142-146`) walks stored chats and replays each message into Chatwoot via the same `CreateMessage` REST path (`gowa:src/infrastructure/chatwoot/sync.go:67-275`). `GET /chatwoot/sync/status` returns progress.

So the GoWA-native model needs a **standard Chatwoot inbox that fires `message_created` webhooks at GoWA's `/chatwoot/webhook`** for the return leg — not just an API the gateway pushes into.

## B. chatwoot-app side

The fork has a **fully bespoke, fork-owned `whatsapp_web` channel** using the **legacy custom-webhook model** — the inverse direction of GoWA-native inbound. GoWA pushes WhatsApp events to a fork-defined Rails endpoint; the fork transforms a custom payload shape into Chatwoot's internal model.

- **Provider registration is fork-owned via prepend overlay**, not a rebase-surface edit. `Fork::Channel::Whatsapp` adds `whatsapp_web?` + `provider_service` returning `Whatsapp::Providers::WhatsappWebService` (`cw:fork/app/models/fork/channel/whatsapp.rb:5-26`). The frozen `PROVIDERS` constant + its inclusion validator are surgically swapped to add `whatsapp_web` at boot (`cw:fork/config/initializers/00_prepend_overlays.rb:29-53`).
- **Inbound webhook controller:** `POST /webhooks/whatsapp_web` → `Webhooks::WhatsappWebController#process_payload` (`cw:fork/config/routes.rb:37`, `cw:fork/app/controllers/webhooks/whatsapp_web_controller.rb:7-32`). Routed via the fork's appended route block (`cw:fork/config/routes.rb:11`). Resolves the channel by `params[:device_id]`, enqueues `Webhooks::WhatsappWebEventsJob`.
- **HMAC verification is here.** `verify_webhook_secret` computes `sha256=<hex_hmac_sha256(raw_post, WHATSAPP_WEB_WEBHOOK_SECRET)>` and `secure_compare`s it to header `X-Hub-Signature-256` (`cw:fork/app/controllers/webhooks/whatsapp_web_controller.rb:55-68`). Fail-closed in prod when secret/API-URL unset (`:40-49`).
- **Custom payload shape parser:** `Whatsapp::IncomingMessageWhatsappWebService` (`cw:fork/app/services/whatsapp/incoming_message_whatsapp_web_service.rb`) parses GoWA's fork-flavoured shape: top-level `event` (`message`/`message.ack`/`message.reaction`/`history_sync_complete`/`message.revoked`/`message.edited`) + `payload` with `chat_id`, `from`, `from_name`, `chat_name`, `contact_name`, `is_from_me`, `from_lid`, `replied_to_id`, media objects (`:47-71,537-607`). This is the fork's custom shape, **not** Chatwoot's standard webhook shape.
- **Outbound (Chatwoot agent reply→WhatsApp)** does NOT use GoWA's `/chatwoot/webhook`. The fork's `provider_service` POSTs directly to GoWA's send endpoints — `POST {WHATSAPP_WEB_API_URL}/send/message`, `/send/image|video|audio|file`, header `X-Device-Id` (`cw:fork/app/services/whatsapp/providers/whatsapp_web_service.rb:34-35,205-228,230-377`).
- **History sync is fork-pulled, not GoWA-pushed.** On `history_sync_complete` the fork *pulls* from GoWA's `/chats`, `/chat/{jid}/messages`, `/message/{id}/download` (`cw:incoming_message_whatsapp_web_service.rb:842-893`, `cw:whatsapp_web_service.rb:129-184`). This is the opposite of GoWA's `/chatwoot/sync` push.
- **Device management** (QR/status/reconnect/logout) — fork calls GoWA `/devices/{id}/login|status|reconnect|logout` (`cw:whatsapp_web_service.rb:42-85`); exposed at `cw:fork/app/controllers/api/v1/accounts/whatsapp_web/devices_controller.rb` (`cw:fork/config/routes.rb:20-29`). Orthogonal to the chatwoot integration — GoWA-native has no equivalent.
- **Not Chatwoot's native Agent-Bot / API channel.** It is a fully custom provider grafted onto `Channel::Whatsapp`.

## C. Topology determination

**Model 3 (hybrid) is the model GoWA v8.5 implements.** Strongest single piece of evidence: `gowa:src/ui/rest/chatwoot.go:39-145` — GoWA's `/chatwoot/webhook` explicitly consumes Chatwoot's standard `message_created`/`outgoing` event and dispatches it to WhatsApp via `SendUsecase`. That is a dedicated *outbound* return leg. Model 2 (retire-ingest) would need no return path; the native code ships one, so the integration is inherently bidirectional.

Today's fork is a **different** Model 3 (custom both ways): inbound = GoWA→fork custom webhook; outbound = fork→GoWA HTTParty. GoWA-native is also Model 3 but with both legs re-homed onto Chatwoot's standard API/webhook surface.

GoWA-native bidirectional flow:

```
INBOUND  WhatsApp → GoWA(whatsmeow) → forwardToChatwoot goroutine
         → POST {ChatwootURL}/api/v1/accounts/{acct}/contacts|conversations|messages
           auth: api_access_token header (Chatwoot API token)        [API-direct PUSH]

OUTBOUND Chatwoot inbox fires message_created (outgoing) webhook
         → POST {GoWA}/chatwoot/webhook  (no auth; basic-auth-excluded)
           body: Chatwoot standard webhook payload
         → GoWA SendUsecase → WhatsApp                               [Chatwoot std webhook]

HISTORY  POST {GoWA}/chatwoot/sync (API-token auth) → GoWA replays
         stored chats into Chatwoot via the same /messages API       [API-direct PUSH]
```

The discriminator answered: yes, Chatwoot must call back into GoWA for agent replies, so the target inbox must be a **standard Chatwoot inbox subscribed to fire `message_created` at GoWA's `/chatwoot/webhook`** — not merely an endpoint GoWA pushes into.

## D. Scoped chatwoot-app plan

**Concrete chatwoot-app-side changes (enumerated, not written):**

1. Stop routing WhatsApp ingest through the fork. Either retire or neuter `cw:fork/app/controllers/webhooks/whatsapp_web_controller.rb`, `cw:fork/app/jobs/webhooks/whatsapp_web_events_job.rb`, `cw:fork/app/services/whatsapp/incoming_message_whatsapp_web_service.rb`, and the `cw:fork/config/routes.rb:37` webhook route. (inference — exact disposition depends on Q1/Q2.)
2. Provision the target Chatwoot inbox as a standard inbox GoWA can push to and that fires `message_created` webhooks back at GoWA's `/chatwoot/webhook` (an API channel / agent-bot inbox, or webhook-enabled). May require re-homing or relabelling existing `provider: whatsapp_web` inboxes (`cw:fork/app/models/fork/channel/whatsapp.rb:6`).
3. Reconfigure the outbound leg. GoWA-native expects Chatwoot's webhook to reach `/chatwoot/webhook`; today the fork pushes outbound via `cw:fork/app/services/whatsapp/providers/whatsapp_web_service.rb:213-228`. Decide whether the fork's `provider_service` send path stays (as the agent-reply transport) or is replaced by Chatwoot's standard webhook firing at GoWA.
4. Keep device-management endpoints working — `cw:fork/.../whatsapp_web/devices_controller.rb` + `cw:whatsapp_web_service.rb:42-85` still need to reach GoWA `/devices/*`; GoWA-native chatwoot has no replacement. These survive the cutover regardless.
5. Configuration: GoWA holds `CHATWOOT_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`, `CHATWOOT_INBOX_ID`, `CHATWOOT_DEVICE_ID` (`gowa:src/infrastructure/chatwoot/client.go:92-106`). These must match the deployed Chatwoot tenant's account/inbox IDs.

**Rebase-surface allowlist risk: ZERO.** The 11-file allowlist (`cw:fork/bin/check-rebase-surface.sh:70-83`) contains no WhatsApp-Web file. The entire channel lives under `fork/**` (excluded pathspec, `:60-68`) and is injected via prepend/route-append. Cutover work is invisible to the rebase audit. This is the one unambiguous good-news constraint.

**Open questions for a chatwoot-app Q-stage:**

1. **Scope-defining:** Accept the feature regression, or port fork-only behaviour onto the API-direct path? GoWA-native lacks: LID phone-discovery (3 scenarios, `cw:incoming_message_whatsapp_web_service.rb:171-353` — GoWA only stuffs `waha_whatsapp_jid` as a flat custom attr, `gowa:client.go:183-186`); status-receipt progression with pessimistic lock (`cw:…:797-830`); reactions stored *on* the original message (`cw:…:140-167`) vs GoWA posting a new "reacted" message (`gowa:webhook_forward.go:242-264`); group-sender contacts + sender-avatar sync; two-phase history-sync contact normalization (`cw:…:842-893`).
2. **Migration:** Re-home existing `provider: whatsapp_web` inboxes/conversations/contacts to a new API-channel inbox, or keep `whatsapp_web` as a vestigial provider label and only rewire ingest?
3. **Device management post-cutover:** Where do QR/status/logout live (`cw:whatsapp_web_service.rb:43-85`)? GoWA-native chatwoot exposes none of these; the fork's HTTParty calls to GoWA `/devices/*` must keep working.
4. Is GoWA's `CHATWOOT_ACCOUNT_ID`/`CHATWOOT_INBOX_ID` config aligned with the deployed Chatwoot tenant, and is the target inbox already a standard/API inbox or does it need creating?

**Size: LARGE (>3 days, architectural).** Driver is **feature parity, not wire protocol**. The protocol swap alone (Model-1-style parser change) would be small, but GoWA-native push does not replicate the fork's LID resolution, reaction-on-message, status-progression locking, group-sender contacts, or two-phase history normalization. Closing that regression set — plus an inbox/provider migration for existing data and preserving the orthogonal device-management path — is the architectural cost. Do not down-scope to medium by ignoring the regression set.
