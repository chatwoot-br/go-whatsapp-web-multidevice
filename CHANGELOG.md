# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v8.10.0+1] - 2026-07-10

### Upstream Sync
- **Synced the fork onto upstream `v8.10.0`** (latest upstream release tag) from the `v8.9.0` base. Single merge commit, no Phase B: upstream's 7-commit unreleased tail (per-device login endpoints, `WHATSAPP_PROXY` #664, `WHATSAPP_WEBHOOK_IGNORE_JIDS` #736, send-link fix #661, chat composer UI, forward-by-ID #755, a newer whatsmeow bump) rides the next sync. whatsmeow `v0.0.0-20260622` → `v0.0.0-20260630`. Unlike v8.9.0, webhook-path changes landed **inside** the tag, so the contract-drift check ran in Phase A: **0 breaking / 2 benign-behavioral / HMAC stable** — the global-webhook path the Chatwoot controller consumes is byte-compatible (nil device config ⇒ old whitelist semantics, global URLs, global secret). See `.workstreams/2026-07-10-upstream-v8.10-sync/`.

### Added (from upstream)
- **Per-device webhooks (#671 + review fixes)** — `devices` table gains `webhook_url`/`webhook_secret`/`webhook_events`/`webhook_insecure_skip_verify` (append-only migrations 31–34); `POST /devices` accepts webhook fields, `PATCH`/`GET /devices/:id/webhook` manage them; delivery resolves per-device config by JID with fallback to the global webhook. **Fleet note:** a device webhook *replaces* the global URL for that device (documented upstream semantics) — fleet instances are unaffected until a device row sets `webhook_url`.
- **Passkey pairing (#754)** — `PairPasskey*` events handled + websocket broadcasts; `GET /app/passkey`, `POST /app/passkey/response|confirm` endpoints.
- **MCP send/manipulation tools (#722)** + migration of MCP handlers to mcp-go request helpers; incoming location messages persisted.
- **Newsletter latest-messages endpoint (#749)**; keep-slot logout (#728 — our upstreamed fix): remote logout disconnects and preserves the device slot + chat history (the old fork-side `TruncateAllDataWithLogging("REMOTE_LOGOUT")` wipe is gone **by design**); broadcast/status messages never reach webhooks regardless of Chatwoot (e4c62f8); `POST /send/file` no longer panics on non-multipart bodies (#748); chatwoot `@newsletter` JID guard (native module remains dormant).

### Preserved (fork features)
- BR ninth-digit phone stack, LID dedup + `history_sync_complete`, full history sync + `ON_DEMAND`, SOCKS/HTTP/HTTPS proxy, `chat_name`/`sender_name` webhook fields (repo-threaded `forwardMessageToWebhook` kept through upstream's forward-path restructure), `normalizeLIDBounded` 30s LID deadline, `InitWaDB` bounded retry, receipt `Device == 0` guard, detached event-dispatcher context, `ChatStorageMaxOpenConns = 1`. GoWA-native Chatwoot stays dormant (`CHATWOOT_ENABLED=false`). Post-merge audit (§7.5) clean: device-scoping sweep, default-flip diff, whatsmeow-workaround check — details in `.workstreams/2026-07-10-upstream-v8.10-sync/03-fork-delta-review.md`.

### Fixed (xhigh code review of the merge; first four are upstream-forwardable)
- **device: `POST /devices` with webhook fields and an auto-generated id no longer 500s and orphans the slot** — the webhook config was persisted under the caller-supplied (empty) id instead of the created `inst.ID()`.
- **whatsapp: per-device webhook config lookups are TTL-cached (30s)** — the merge ran an uncached `devices`-table SELECT per forwarded event (every message, receipt, typing burst) on the `MaxOpenConns=1` chatstorage pool, head-of-line queuing against message writes; cache stores nil results too, is invalidated on webhook-config writes, and is bypassed under the test seam.
- **whatsapp: logout/delete now remove *all* whatsmeow store rows for the account** — `deleteStoreRowsForJID` stopped after the first NonAD match, so a stale same-account row survived and `LoadExistingDevices` resurrected the session on restart.
- **whatsapp: `DELETE /devices/{jid}` on a uuid-keyed slot purges for real** — `PurgeDevice` gained `keepSlotLogout`'s stale-id JID fallback and cleans up under the resolved registry key (previously: silent success leaving store rows + the live slot).
- **utils: disconnected clients get a clean 401 on group/newsletter endpoints** — `MustLogin` now gates before `ValidateAndNormalizeJID`'s non-user-JID early return (upstream `ValidateJidWithLogin` parity; previously a raw 500 from deep inside whatsmeow).
- **whatsapp: purge deletes chat data under both the slot id and the NonAD JID** (sweep pass on the fix above — chat/message rows are JID-keyed while the devices row is slot-keyed; purging a paired uuid slot previously left the tenant's conversation history in the shared DB).
- **mcp: tool-handler panics no longer kill the process** — `server.WithRecovery()` added; the usecase layer panics on auth failures (`MustLogin`), which only REST's Recovery middleware caught, so one disconnected-client tool call in MCP mode took down every device session.

### Fixed (Codex adversarial review + issue #10)
- **Logout-then-DELETE purges the retained history** — keep-slot logout records the JID it clears in a new `devices.last_jid` column (append-only migration 35), and `PurgeDevice` deletes chat data under the slot id, live jid, *and* last_jid; previously a DELETE after logout reported success while leaving the JID-scoped chats/messages orphaned in the shared DB. Device-record getters now share one full column list (`ListDeviceRecords`/`GetDeviceRecord` used to zero the webhook fields that `GetDeviceRecordByJID` hydrated).
- **No-consumer deployments no longer download media** — a `hasAnyWebhookConsumer` gate (global webhook ∨ Chatwoot ∨ TTL-cached device-webhook scan) runs before webhook payload construction, which is the only downloader of non-image media; also kills the per-event "Forwarding to 0 webhooks" log.
- **Device webhook lookup errors serve the last-known config** (even past its TTL) instead of silently diverting a device with a webhook override to the global URL on a transient storage error; fall-back-to-global remains only when nothing was ever cached.
- **Failed purges are retryable** — `PurgeDevice` keeps the slot and device record when local cleanup fails (they hold the id↔jid mapping a retry needs) and surfaces device-record deletion errors instead of masking them.
- **Remote-logout state reconciliation is deterministic (#10)** — `handleLoggedOut` runs the keep-slot cleanup directly when no callback is wired; `Reconnect` on a logged-out slot returns a typed 401 (`ErrSessionDeleted`) pointing at the re-pair flow instead of a raw 500; boot auto-connect logs logged-out slots as "awaiting re-pair" info instead of warning every restart.
- Merge adaptations: upstream tests updated to fork signatures (`handleWebhookForward` 4-arg, `submitWebhookFn` +`DeviceWebhookConfig`); `newsletter.GetMessages` follows the fork's `ValidateAndNormalizeJID` caller convention.

### Fixed (Codex review round 2 — findings in fork-authored code)
- **The media gate is scoped to the device, not the fleet** — `hasAnyWebhookConsumer` answered "a consumer exists" for *every* device as soon as *any* device owned a per-device webhook, so devices with no destination of their own still entered payload construction and downloaded media to disk for an event that was then discarded. `hasWebhookConsumerForDevice` now resolves the current device's own destinations (global ∨ Chatwoot ∨ its own webhook), failing open on a lookup error.
- **A failed logout is retryable** — keep-slot logout cleared the in-memory JID even when the whatsmeow store cleanup failed, so the retry arrived with no JID, deleted nothing, and reported success — leaving the orphan row that `LoadExistingDevices` resurrects on restart. The retry now recovers the identity from `last_jid`.
- **Re-pairing a slot to a different account no longer strands the old account's history** — `last_jid` holds one identity, so logging out of A, re-pairing to B, then logging out again overwrote the only pointer to A's retained chat data, which no later `PurgeDevice` could still name. A's data is now purged as the pointer is replaced (unchanged JID = no-op, so logout retries still preserve history).
- **`Reconnect` on a logged-out slot returns the typed 401, not a 500** — `ResetClient` detaches the client entirely, so the logged-out state reaches `ReconnectDevice` as a **nil client**, never as a nil `Store.ID`; the `ErrSessionDeleted` branch added for it was therefore unreachable behind a generic "client not initialized" error.
- **`POST /devices` rolls the slot back when webhook persistence fails** — the slot was left registered behind the returned error, so the same `device_id` then failed with "device already exists" (or leaked an unnameable ghost slot under an auto-generated id) and the create could not be retried without manual cleanup.
- **`GetDeviceRecordByJID` is deterministic** — `WHERE jid = ? LIMIT 1` with no `ORDER BY` could return a legacy config-less duplicate row over the named slot that actually carries the webhook, intermittently diverting a configured device's events to the global webhook. It now prefers the configured row, then the most recently updated.

### Known upstream issues (verified, reported — not fixed here; per-device-webhook users only)
- `PATCH /devices/:id/webhook` has PUT semantics (plain fields — omitting `webhook_secret`/`webhook_events` wipes them); a URL-less webhook config is accepted and echoed but never applied; device `webhook_insecure_skip_verify=false` cannot re-enable TLS verification when the global flag is true (docs promise "override"); an empty device `webhook_events` falls back to the global whitelist while openapi.yaml promises "all events"; an empty device `webhook_secret` signs tenant-URL deliveries with the global secret; MCP mode never drains `websocket.Broadcast`, so passkey/pair events stall whatsmeow's handler queue ~5min each (pre-existing, more send sites now); the UI passkey confirm posts to the currently-selected device rather than the broadcast's `device_id`; with **no** webhook/Chatwoot consumers configured, every media message is still downloaded to `statics/media` for a payload that is then discarded; the four new MCP send tools (video/document/audio/poll) skip `SanitizePhone`, so bare group ids parse as user JIDs. (Two of these are fixed fork-side and remain open upstream: the no-consumer media download — see the device-scoped consumer gate — and the `GetDeviceRecordByJID` duplicate-row ordering.)

## [v8.9.0+1] - 2026-07-02

### Upstream Sync
- **Synced the fork onto upstream `v8.9.0`** (latest upstream release tag) from the `v8.7.0` base. Single merge commit, no Phase B: upstream's 2-commit unreleased tail (a whatsmeow bump + #748 non-multipart `/send/file` panic fix) is left for the next sync. whatsmeow `v0.0.0-20260609` → `v0.0.0-20260622`; `golang.org/x/net` v0.55.0 → v0.56.0. Contract-drift check clean — upstream touched **no** webhook-forwarding code in this range (0 breaking / 0 behavioral / HMAC stable). See `.workstreams/2026-07-02-upstream-v8.9-sync/`.

### Added (from upstream)
- **Media `direct_path` persistence (#731)** — new `messages.direct_path` column (append-only migration) threaded through chatstorage; `ExtractMediaInfo` returns it and new `ResolveMediaDirectPath`/`BuildDownloadableMessage` helpers use it so media stays downloadable after WhatsApp URL expiry. Storage-internal; not part of any webhook payload.
- **Call-reject API (#735)** — `POST /call/reject` (`ui/rest/call.go`, `usecase/call.go`, `domains/call/`, `CallReject.js` view) rejects a still-ringing call using `call_id`/`from` from the existing `call.offer` webhook. Inbound-only; emits no webhook.

### Changed (from upstream)
- **WhatsApp error 463 is no longer retried (#708)** — upstream deleted `send_retry.go`/`send_retry_test.go` (the fork never modified them) and added `reachout_error.go`: a 463 "reachout" send failure is surfaced honestly instead of burning blind retries.
- Random API send timeouts under heavy incoming message load fixed (#732); `GET` chat messages returns an empty result instead of HTTP 500 when the chat row is absent (#740); native-chatwoot `pgimport` uuid-cast fix (#724 — module remains dormant).

### Preserved (fork features)
- BR ninth-digit phone probes (`brPhoneCandidates`/`probeBRPhone` wired in `ValidateAndNormalizeJID`), LID dedup + `history_sync_complete`, full history sync + `ON_DEMAND`, SOCKS/HTTP/HTTPS proxy, `chat_name`/`sender_name` webhook fields, `InitWaDB` bounded retry. GoWA-native Chatwoot module stays dormant (`CHATWOOT_ENABLED=false`); the fork's integration remains the active path. (The dormant info cache is **removed** this release — see below.)
- Provenance correction: HMAC `X-Hub-Signature-256` signing/verification is **upstream-owned** as of this base (byte-identical in both trees) — no longer a fork-carried feature; the fork retains only its extra test coverage. Full fork-delta review vs v8.9.0 (33 keep / 2 drop-candidates / 3 converged): `.workstreams/2026-07-02-upstream-v8.9-sync/03-fork-delta-review.md`.

### Fixed (Codex review on the sync PR; first three are upstream-forwardable)
- **whatsapp: event handlers no longer run under a cancelled request context.** `EnsureClient`/`InitWaCLI` registered the WhatsApp event handler with the caller's context; when reached from `/app/login`, every later event (history sync, LID resolution, storage writes) ran under the already-cancelled request context — LID chats stored unresolved for REST-paired devices. Both registration sites now detach with `context.WithoutCancel`.
- **chat: `GET /chat/:jid/messages` returns stored rows when the chat record is absent.** Upstream #740's `chat == nil` early-return hid message rows that exist before the chat upsert — the Chatwoot history import would silently import 0 messages. Falls through to the message query and synthesizes minimal chat metadata; regression test added.
- **chatwoot: direct_path-only media no longer fails the REST import.** The media pre-pass accepted rows with only `direct_path`+`media_key` (upstream #731) but the download gate still required a non-empty `url`, guaranteeing "required media attachment is unavailable"; the gate now reuses the same predicate. (Dormant module in fork deployments.)
- **call: malformed `caller_jid` on `POST /call/reject` returns 400, not 500**, and the trimmed request values are the ones actually used.

### Fork-surface reduction (review-driven, behavior-neutral)
- **Reverted the LID caller swap.** Restored upstream's `NormalizeJIDFromLID(ctx, …)` wrapper and its exact call-site text at the ~10 sites the fork had pointed at the byte-identical `utils.ResolveLIDToPhone`, and dropped the fork's ctx-less `NormalizeJIDFromLIDWithContext` variant. `jid_utils.go` and six `event_*.go` files are byte-identical to upstream again, shrinking every future sync's conflict surface. (The 30s per-lookup bound that variant provided is restored on the history-sync paths by `normalizeLIDBounded` — see the code-review fixes below.)
- **Removed the dormant short-term info cache** (`infrastructure/whatsapp/info_cache.go`, `pkg/cache/`) — shipped dormant in the v8.5 sync and never gained a caller; recoverable from git history if ever wired for real.
- **Documented ON_DEMAND history sync as handler-only**: nothing calls `BuildHistorySyncRequest`, so the handler serves only rare unsolicited syncs; wiring a backfill endpoint is a deliberate future feature, not merge fallout.

### Fixed (xhigh code review of the merge)
- **Restored `SetMaxOpenConns` default to 1 for chat storage.** Upstream #732 changed the default to 5, but the fork's emulated upserts (`StoreChat`/`StoreMessage`/`StoreReaction` do UPDATE-then-INSERT with no transaction or `ON CONFLICT`) and `MergeLIDChat`'s tx-scoped reads are only safe under a single connection — the `MergeLIDChat` comment already documented that invariant. At 5 connections a burst of concurrent sends to the same not-yet-stored recipient could both fall through UPDATE and both INSERT, dropping the loser's row with only a warning. Kept the `CHAT_STORAGE_MAX_OPEN_CONNS` override for when the upserts are made atomic.
- **Device-scoped three chat/message reads that leaked across devices** on a shared chat-storage DB: `GetChatMessages` pagination total (`GetChatMessageCount` → `GetChatMessageCountByDevice`) and sender-name lookup (`GetChat` → `GetChatByDevice`), and `DownloadMedia`'s message fetch (`GetMessageByID` → `GetMessageByIDAndDevice`, closing a cross-device media-download hole). Also deduped the per-message sender lookup within a page.
- **Restored the 30s bound on history-sync LID resolution.** New `normalizeLIDBounded` wraps each `@lid`→phone lookup in a per-call 30s deadline at the history-sync call sites (the paths run under a detached or `context.Background()` context with no deadline), so a slow/contended LID store lookup can no longer stall the sync pipeline indefinitely — while keeping `jid_utils.go` byte-identical to upstream.
- **Moved the event-handler context detach into the shared `handler()` dispatcher** instead of the two `AddEventHandler` registration sites, so a future third registration site can't reintroduce request-scoped cancellation.
- **`DownloadMedia` now requires `MediaKey` up front**, returning a clear "no downloadable media" error instead of failing deep in whatsmeow with "invalid media hmac" for a `direct_path`-only, key-less row.
- **Extracted `chatInfoFromEntity`** shared by `ListChats` and `GetChatMessages` (was two drifting inline literals); dropped the redundant second trim in `ValidateRejectCall` (the usecase already trims what it passes on).

## [v8.7.0+2] - 2026-06-12

### Fixed
- **whatsapp: outbound sends to Brazilian contacts registered under a different ninth-digit form failed with "Phone X is not on WhatsApp" (#8).** `ValidateAndNormalizeJID` stripped the BR mobile 9th digit and probed ONLY the resulting 12-digit form, so a contact whose WhatsApp account is registered under the 13-digit (with-9) form got an authoritative `IsIn=false` and the send was rejected (with `WHATSAPP_ACCOUNT_VALIDATION` on). It now probes BOTH ninth-digit forms in a single USync call (`brPhoneCandidates` + `probeBRPhone`) and resolves to whichever WhatsApp confirms, preferring the as-dialed form; both directions (dialed-13/registered-12 and the inverse) are covered, and the `/user/check` endpoint + group-add guard inherit it. Verified live: `+5511945590462` (with 9) is on WhatsApp, `+551145590462` (without) is not.

### Safety / correctness (review-driven hardening of the above)
- **Ninth-digit sibling is mobile-gated.** The sibling form is generated only for mobile-range local numbers (8-digit local starting 6-9; `isBRMobileLocal`). A 12-digit landline's "+9" sibling is a *different* subscriber's mobile, so an ungated probe could confirm — or silently route a send to — a stranger; gating eliminates that misroute. Trade-off: a mobile whose local part starts 2-5 won't auto-resolve via its sibling (its as-dialed form is still probed).
- **Partial USync responses no longer hard-fail a valid recipient.** With >1 candidate, a non-empty response that omits a queried form (whatsmeow emits one entry per server-resolved `<user>` node, with no per-query echo guarantee) is treated as inconclusive (retry → ambiguous → fall open), not an authoritative negative.
- **Group-add uses WhatsApp's confirmed canonical JID.** `participantToJID` routes through `ValidateAndNormalizeJID` (probes both forms, returns the canonical JID) instead of re-parsing the raw dialed string, so a contact registered under the sibling form is added under the form WhatsApp holds. It also guards connectivity/login up front (clean error instead of a `MustLogin` panic on the join-request path) and surfaces the real resolution error rather than a blanket "user not registered".

## [v8.7.0+1] - 2026-06-11

### Upstream Sync
- **Synced the fork onto upstream `v8.7.0`** (latest upstream release tag) from the `v8.5.0` base. The merge also pulls upstream's post-`v8.7.0` `main` tail — the still-unreleased work that bumped upstream's in-development version string to `v8.8.0` (not yet a tagged release, so the fork rail follows the `v8.7.0` tag). whatsmeow `v0.0.0-20260513` → `v0.0.0-20260609`; Go 1.25.5; new pure-Go SQLite (`modernc.org/sqlite`, build-tag selected via `pkg/sqlite`); fiber/fasthttp/libsignal bumps. See `.workstreams/2026-06-11-upstream-v8.7-sync/`.

### Added (from upstream)
- Message reactions: persisted, mapped into `GetChatMessages`, emitted as `message.reaction` webhooks, and stored from history sync.
- `SecretEncryptedMessage{MESSAGE_EDIT}` decryption for modern LID-migrated clients.
- Label appstate → webhook forwarding, scheduled presence pulse, WhatsApp 463 send-retry mitigation, quoted media replies, ARMv7 / pure-Go SQLite build path.
- `session_id` top-level webhook field — emitted when the device JID maps to a registered session (additive; chatwoot-app ignores unknown keys).

### Changed
- **Chatwoot contact custom attribute `waha_whatsapp_jid` → `gowa_whatsapp_jid`** (upstream rebrand). New contacts are written with `gowa_whatsapp_jid`; `FindContactByIdentifier` and the inbound agent-reply route **read `waha_whatsapp_jid` as a back-compat fallback**, so contacts created before this release keep routing — no data migration required.
- **`FindOrCreateContact` now preserves an existing non-empty 1:1 contact name** (fills blank names only; group subjects still refresh) instead of always overwriting — adopts upstream #675/#688 (don't clobber a saved/agent-edited name with a phone number).

### Preserved (fork features)
- BR phone normalization, LID dedup + `history_sync_complete`, full history sync + `ON_DEMAND`, info cache, SOCKS/HTTP/HTTPS proxy, `chat_name`/`sender_name` webhook fields + HMAC `X-Hub-Signature-256`, `InitWaDB` bounded retry. GoWA-native Chatwoot module stays dormant (`CHATWOOT_ENABLED=false`); the fork's integration remains the active path.

## [v8.5.0+5] - 2026-06-05

### Fixed
- **whatsapp: `InitWaDB` panicked (no retry) on transient DB unavailability at startup, crash-looping on DB blips (#5).** Opening the WhatsApp store DB called `sqlstore.New` exactly once and turned any error into a `panic`, so a *transient* transport error (e.g. `connection refused` while an external Postgres restarts / fails over / a node blips) was fatal: the process exited and the container crash-looped until the DB happened to be reachable at the instant of a (re)start. Each crash also forced a fresh WhatsApp reconnect whose degraded post-reconnect USync window made send-time `IsOnWhatsApp` validation (#3/#4) more likely to return inconclusive results. `InitWaDB` now resolves the driver first and **fails fast on an unsupported/unknown DB type** (a permanent config error retrying can't fix), then opens the store with **bounded, context-aware retries** (`openDBWithRetry`: 6 attempts, 8s backoff, each attempt capped by a 10s connect timeout so a black-holed dial can't hang startup) — ≈40s for the common `connection refused` fast-fail, bounded ~100s worst case. It now returns an error instead of panicking, so `cmd/root.go` exits cleanly via `logrus.Fatalf` (matching the adjacent chat-storage init). Credentials in an unknown-type `DBURI` are redacted before they reach logs. Logic sits behind an injectable `dbOpener` seam with unit tests (`database_test.go`) covering retry, ctx-cancel, the per-attempt timeout, fail-fast, and credential redaction.

## [v8.5.0+4] - 2026-06-03

### Fixed
- **whatsapp: `WHATSAPP_ACCOUNT_VALIDATION=true` caused intermittent false-negative "Phone X is not on WhatsApp" send failures (#3).** `ValidateAndNormalizeJID` treated a single, un-retried `client.IsOnWhatsApp` USync probe as authoritative. WhatsApp's USync is non-deterministic (throttling, post-pairing app-state sync gaps), so a genuinely-registered number intermittently came back empty or `IsIn=false`, and the send failed permanently with a misleading terminal error (surfaced in Chatwoot as "Falha ao enviar"). The probe is now classified as **positive / confirmed-negative / ambiguous**, retried with bounded attempts under a single total deadline (`onWhatsAppTotalTimeout`), and an **ambiguous** result (empty response or transport error) falls through to the BR-normalized JID and sends instead of hard-failing — even when validation is on. Only an authoritative `IsIn=false` (non-empty response) is still rejected. The fall-through JID is built from the BR 9th-digit-stripped, E.164-normalized phone so delivery to BR mobiles is preserved. `IsOnWhatsapp` (backing the `/user/check` API and the group-add guard) stays honest: ambiguous => `false`, never fall-open. Logic extracted behind an `onWhatsAppProber` seam (`probeOnWhatsApp` / `resolveProbeOutcome` / `resolveUserJID` / `isOnWhatsApp`) with table-driven unit tests, including the bounded-deadline cutoff and the BR fall-open path.

## [v8.5.0+3] - 2026-05-22

### Fixed
- **rest: `GET /chat/:chat_jid/messages` returned HTTP 500 ("chat with JID ... not found") for every request.** Fiber's `UnescapePath` defaults to `false`, so the percent-encoded chat JID sent by the Chatwoot history-sync client (`...%40s.whatsapp.net`) reached the handler still encoded. The chatstorage lookup (`GetChatByDevice`) is an exact string match, so it never matched the stored JID (`...@s.whatsapp.net`); `chat == nil` raised an error that `PanicIfNeeded` converted to a recovered panic / HTTP 500. Every per-chat fetch during a Chatwoot history sync failed, so the sync completed having imported 0 messages. Enabled `UnescapePath` so path params are percent-decoded before routing — also fixes the same latent bug on `/chat/:chat_jid/{pin,disappearing,archive}`. Added regression test `TestRestFiberConfigDecodesEncodedChatJID` that drives the production fiber config.

## [v8.5.0+2] - 2026-05-21

### Fixed
- **chatstorage: `MergeLIDChat` deadlocked under `MaxOpenConns(1)`.** The transaction opened by `MergeLIDChat` held the only connection in the chatstorage pool (set by `cmd/root.go:initChatStorage`), then called `r.GetChatByDevice` twice — those helpers issue `r.db.QueryRow`, which requested a second connection and waited forever. After every history sync the fork-only `deduplicateLIDChats` goroutine triggers this path; if any `@lid` chat existed, the deadlock froze every subsequent `CreateMessage` from incoming WhatsApp events, and `handleWebhookForward` never fired (live messages stopped reaching downstream consumers ~5 s after the history-sync debounce). Inlined the two reads as `tx.QueryRow` so the whole transaction stays on the same connection; added a `MaxOpenConns(1)` invariant note in the function header and a regression test (`TestMergeLIDChat_NoDeadlockWithSingleConnPool`) that pins the behavior under the production pool size.

## [v8.5.0+1] - 2026-05-14 (Synced with upstream v8.5.0)

### Upstream Changes
- ~31 whatsmeow protocol updates (security/compatibility patches)
- Native Chatwoot integration: src/infrastructure/chatwoot/{client,sync,sync_test,sync_types,types}.go, src/ui/rest/chatwoot.go, /chatwoot/webhook + /chatwoot/sync* endpoints, 8 CHATWOOT_* env vars
- LID handling improvements: ResolveLIDToPhone/ResolvePhoneToLID primitives, LID-aware auto-reply, group-participants phone fix
- Webhook taxonomy: chat_presence (typing), call.offer (incoming call), contacts_array shape, media captions in payloads, is_from_me top-level field
- feat: healthcheck endpoint, GIF playback, document thumbnails, CTWA Meta Ads referral support, ghost mentions, archived chats filtering
- fix: Docker permission readonly DB on group messages, document thumbnail security, audio extension test parser
- chore: Go 1.25 / Alpine 3.23, dependency updates

Full upstream commit log: git log v8.1.2..v8.5.0

### Fork Changes
- slice 1: reset to upstream/v8.5.0 + reapply release rail (Helm chart, 4 CI workflows, Dockerfile mailcap)
- slice 2: BR phone normalization layer (src/pkg/utils/phone_br.go) + 39 caller sweep across src/usecase/{send,group,message,chat,user,newsletter}.go (preserves v8.1.0+7 ValidateAndNormalizeJID behavior on upstream baseline)
- slice 3: LID dedup + history_sync_complete dispatch (preserves v8.1.0+6 deduplicateLIDChats post-history-sync pass + MergeLIDChat/GetLIDChats chatstorage primitives + NormalizeJIDFromLIDWithContext 30s-timeout variant; new file forward_history_sync.go scopes the fork-specific event)
- slice 6: fork-only delta sweep (proxy support v8.1.0+3, info-request cache v8.1.0+2, S3 image-extension fix v8.1.0+5; OQ8 device_manager.go own-commit due to upstream/any-modernization overlap; audio/PTT v8.1.0+1 and APP_BASE_PATH v8.1.0+1 subsumed by upstream)
- slice 4: chatwoot lockstep cutover gateway-side wiring (8 CHATWOOT_* env vars in Helm + configmap; /chatwoot/webhook route order verified; chatwoot-app Rails cutover documented as separate-repo follow-up)
- slice 5: webhook taxonomy + env audit (docs/webhook-payload.md union of upstream events + fork's history_sync_complete; WHATSAPP_WEBHOOK_INCLUDE_OUTGOING marked deprecated; is_from_me echo-suppression documented)
- fix(webhook): recovered chat_name + sender_name payload fields (v8.1.0+1) missed during slice 6 sweep

### Known follow-ups (out of scope for this upgrade)
- chatwoot-app Rails-side cutover PR (Channel::Whatsapp::Provider rewire)
- Paired-phone validation: trigger each event from staging phone; confirm receipt at test webhook
- Cleanup: usecase callers of new info_cache helpers (left dormant but build-green per Slice 6 agent note)

## [v8.1.2+1] - 2026-01-26 (Synced with upstream v8.1.2)

### Upstream Changes
- feat: add webhook events for newsletters and group.joined
- fix: react to other users' messages by looking up IsFromMe from database (#535)
- fix: webhook event whitelist filtering for groups and proper event names (#539)
- fix(security): prevent cross-device data leak in chat message queries (#525)
- fix(device): sort device list by created_at for stable UI ordering (#528)
- fix: store phone-sent messages in chat history (issue #526) (#530)
- chore: update dependencies (golang.org/x/text to v0.33.0, app version to v8.1.2)

### Fork Changes
- chore: update whatsmeow to latest (v0.0.0-20260126173513-4dbbef8d4d4a)
- fix(docker): add mailcap package for MIME types database

## [v8.1.0+7] - 2026-01-20

### Fixed
- fix(utils): normalize Brazilian phone numbers to prevent duplicate contacts
  - Add ValidateAndNormalizeJID function that handles Brazilian 9-digit mobile normalization
  - Update all callers across send, chat, group, message, newsletter, and user usecases
  - Mark ValidateJidWithLogin as deprecated in favor of the new function

## [v8.1.0+6] - 2026-01-18

### Fixed
- fix(history-sync): resolve LID duplicate chats and context cancellation
  - Add dedicated context with 5s timeout for LID resolution to prevent context cancellation errors
  - Add NormalizeJIDFromLIDWithContext helper for isolated LID lookups
  - Add MergeLIDChat to chatstorage for deduplicating chats with same sender but different JID formats
  - Add post-sync deduplication to merge LID chats after history sync

## [v8.1.0+5] - 2026-01-17

### Fixed
- fix(utils): derive image extension from Content-Type for S3 URLs

## [v8.1.0+4] - 2026-01-15

### Added
- feat(whatsapp): enable full history sync and ON_DEMAND capability
- feat(whatsapp): handle unavailable messages from linked devices
- feat(whatsapp): process ON_DEMAND history sync responses
- feat: add logs directory to .gitignore and create .keep file

### Fixed
- fix(whatsapp): normalize chat_id from LID to phone number in webhook

### Changed
- chore: update dependencies for go.mau.fi/whatsmeow and golang.org/x packages

## [v8.1.0+3] - 2026-01-13

### Added
- feat(proxy): add SOCKS5/HTTP/HTTPS proxy support for WhatsApp connections
- feat(proxy): display external proxy IP in device card UI

### Fixed
- fix(webhook): include caption in payload when auto-downloading media (image, video, video_note)

## [v8.1.0+2] - 2026-01-08

### Added
- feat(cache): add short-term caching for info requests
- feat(webhook): update events list to include history_sync_complete and improve documentation

### Fixed
- fix(cache): cache error responses to prevent repeated API calls
- fix(send): use LID for message delivery with targeted approach
- Various CI workflow fixes for tag patterns and multi-arch builds

### Changed
- refactor(workflow): trigger Helm chart release on version tags only

## [v8.1.0+1] - 2025-01-07

### Added
- feat(helm): add gowa Helm chart for Kubernetes deployment
- feat(webhook): add chat_name to outgoing message payload
- feat(chat): add sender_name field for group message contacts
- feat(whatsapp): add history sync webhook notification
- feat(audio): add OGG Opus conversion for PTT voice notes
- feat(whatsapp): include is_from_me in webhook payload
- feat: add multi-device support guide documentation
- feat: add waveform generation for audio messages
- feat: enhance audio handling with MIME type resolution and duration retrieval

### Fixed
- fix(whatsapp): debounce history sync webhook to wait for all events
- fix(login): use background context for QR channel
- fix(device.go): Fix DeviceMiddleware to allow if APP_BASE_PATH is changed

### Changed
- Updated GitHub Actions workflows to support fork versioning (v8.1.0+1 format)
- Added chart-releaser workflow for Helm chart releases

