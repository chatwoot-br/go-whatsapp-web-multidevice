# P - Plan (Codex draft)

QRSPI P stage for syncing chatwoot-br GoWA `v8.1.2+1` with upstream
`aldinokemal/go-whatsapp-web-multidevice` `v8.5.0`.

Assumptions from Q/D/S: reset+reapply; adopt+migrate native Chatwoot; layer BR
phone rules; all-in upstream; lockstep chatwoot-app cutover; release tag
`v8.5.0+1`; `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` is deprecated/dead, not
removed; the gateway has 8 `CHATWOOT_*` env vars.
## 1. Slice list
### Slice 0 - Reset baseline and release rail
**Goal:** Establish upstream `v8.5.0` as the baseline while restoring the fork's
packaging rail so later slices can ship.
**Files touched:** `CHANGELOG.md`; `.github/workflows/build-docker-image.yaml`;
`.github/workflows/release.yml`; `.github/workflows/chart-releaser.yaml`;
`.github/workflows/set-latest-tag.yaml`; `charts/gowa/{Chart.yaml,values.yaml,templates/,README.md}`;
`docker/golang.Dockerfile`; `src/go.mod`; `src/go.sum`.
**Steps:**
- `git fetch upstream --tags`
- Confirm R refs: `git rev-parse upstream/main v8.5.0 main v8.1.2+1`.
- `git tag -a pre-upgrade-snapshot-2026-05-14 main -m "pre v8.5 sync snapshot"` if absent.
- `git switch -c upgrade/v8.5.0-sync main`; then `git reset --hard upstream/main`.
- Reapply the fork release rail: Docker `+` to `-` tag rewrite, `release.yml`,
  chart releaser, manual latest gate, Helm chart, `mailcap`, and `v8.5.0+1`
  CHANGELOG skeleton.
- Run `git diff --name-only upstream/main...HEAD` and `cd src && go test ./...`.
**Checkpoint:** Upstream `v8.5.0` plus fork packaging builds, tests, and can
publish `v8.5.0+1` as OCI tag `v8.5.0-1`.
**Rollback:** Reset+reapply makes per-slice rollback non-trivial. Tag this
checkpoint, for example `slice-0-v8.5.0-release-rail`; rollback is resetting to
`upstream/main` and replaying Slice 0, or deploying `pre-upgrade-snapshot-2026-05-14`.
### Slice 1 - BR phone normalization path
**Goal:** Reapply BR 9-digit normalization over upstream phone helpers and prove
send/chat/group/message routes canonical JIDs.
**Files touched:** `src/pkg/utils/phone_br.go`; `src/pkg/utils/phone_br_test.go`;
`src/pkg/utils/whatsapp.go`; `src/pkg/utils/general.go`;
`src/pkg/utils/whatsapp_test.go`; `src/pkg/utils/general_test.go`;
`src/usecase/{send,message,user,group,chat,newsletter}.go`.
**Steps:**
- Add `phone_br.go` with the S-locked entry point
  `ValidateAndNormalizeJID(client *whatsmeow.Client, jid string) (types.JID, error)`.
- Keep upstream `phone.go` unmodified; call upstream phone helpers from the BR layer.
- Move BR fixtures from `jid_utils_test.go`, `general_test.go`, and
  `whatsapp_test.go` into `phone_br_test.go` per D-review OQ10.
- Update the 45 S-listed usecase call sites to the same call shape:
  `utils.ValidateAndNormalizeJID(client, request.Phone)`.
- Run `rg -n "ValidateAndNormalizeJID|ValidateJidWithLogin" src/usecase src/pkg`
  and `cd src && go test ./pkg/utils ./usecase/...`.
**Checkpoint:** BR fixtures pass, non-BR numbers preserve upstream behavior, and
a paired phone sends to a BR mobile number without duplicate contact creation.
**Rollback:** Return to the Slice 0 checkpoint. If later slices depend on the
new call-site shape, reset to Slice 0 and replay accepted slices rather than
reverting only the utility file.
### Slice 2 - History sync completion and LID dedup
**Goal:** Preserve fork history-sync completion signaling and post-sync LID chat
dedup on upstream's v8.5.0 webhook/chatstorage base.
**Files touched:** `src/infrastructure/whatsapp/history_sync.go`;
`src/infrastructure/whatsapp/forward_history_sync.go`;
`src/infrastructure/whatsapp/jid_utils.go`;
`src/infrastructure/chatstorage/sqlite_repository.go`;
`src/infrastructure/chatstorage/device_repository.go`;
`src/infrastructure/whatsapp/chatstorage_wrapper.go`;
`src/domains/chatstorage/interfaces.go`; `src/.env.example`;
`docs/webhook-payload.md`; `readme.md`.
**Steps:**
- Reapply `deduplicateLIDChats`, `NormalizeJIDFromLIDWithContext`,
  `NormalizeJIDFromLID`, `MergeLIDChat`, and `GetLIDChats` on S signatures.
- Add `forward_history_sync.go` and call the existing S signature
  `forwardPayloadToConfiguredWebhooks(ctx, payload, eventName) error`.
- Keep the debounced history-sync notification after cached push names and LID dedup.
- Append `history_sync_complete` to `WHATSAPP_WEBHOOK_EVENTS` and document
  `{event, device_id, payload: {sync_type, timestamp}}`.
- Run `rg -n "history_sync_complete|deduplicateLIDChats|MergeLIDChat|GetLIDChats|NormalizeJIDFromLIDWithContext" src docs readme.md`
  and `cd src && go test ./infrastructure/whatsapp ./infrastructure/chatstorage ./domains/chatstorage`.
**Checkpoint:** A paired-phone full history sync merges residual `@lid` chats
and emits exactly one documented `history_sync_complete` webhook.
**Rollback:** Return to Slice 1. This crosses storage, history sync, and webhook
docs; if later slices exist, reset to the checkpoint and replay, rather than
reverting individual storage/webhook files.
### Slice 3 - Fork gateway behavior and overlap sweep
**Goal:** Reapply fork gateway behavior independent of native Chatwoot, then
settle fork-heavier overlap files before the cross-repo cutover.
**Files touched:** `src/infrastructure/whatsapp/info_cache.go`;
`src/pkg/cache/cache.go`; `src/infrastructure/whatsapp/device_instance.go`;
`src/infrastructure/whatsapp/device_manager.go`;
`src/infrastructure/whatsapp/{auto_reply,event_message,event_message_handler,event_group,event_newsletter,init}.go`;
`src/usecase/{app,device,chat,group,message,send,user}.go`; `src/cmd/root.go`;
`src/config/settings.go`; `src/.env.example`; `src/pkg/utils/general.go`;
`docs/issues/*`; `docs/plans/*`; `docs/multi-device-guide.md`.
**Steps:**
- Reapply short-term info-request cache plus error caching.
- Reapply SOCKS5/HTTP/HTTPS proxy support and proxy IP display.
- Reapply OGG Opus PTT conversion, waveform, MIME/duration resolution, media
  caption forwarding, S3 image extension from Content-Type, and `APP_BASE_PATH`.
- Do the OQ8+9 sweep here: inspect `device_manager.go` and
  `usecase/{chat,group,message}.go` after Slices 1-2. Bundle clean reconciliation;
  split only if a real behavior conflict appears.
- Run `git diff --check`, `rg -n "proxy|InfoCache|waveform|Content-Type|APP_BASE_PATH" src`,
  and `cd src && go test ./infrastructure/whatsapp ./pkg/cache ./usecase/...`.
**Checkpoint:** REST smoke tests cover device connect through proxy, info-cache
hits including cached errors, audio/PTT metadata, media caption, and S3 extension.
**Rollback:** Return to Slice 2. If one behavior fails, fix forward or reset and
reapply a smaller subset; avoid partial reverts of `device_manager.go` or usecase
files once Slice 4 begins.
### Slice 4 - Native Chatwoot lockstep cutover
**Goal:** Adopt upstream native Chatwoot integration and cut chatwoot-app over in
the same deployment window.
**Files touched:** `src/infrastructure/chatwoot/{client,types,sync,sync_types,sync_test}.go`;
`src/infrastructure/whatsapp/webhook_forward.go`;
`src/infrastructure/whatsapp/chatwoot_forward_test.go`;
`src/ui/rest/chatwoot.go`; `src/cmd/rest.go`; `src/cmd/root.go`;
`src/config/settings.go`; `src/.env.example`; `docs/chatwoot.md`;
`readme.md`; `charts/gowa/values.yaml`; paired chatwoot-app files are outside
this Go repo and therefore outside S's file inventory.
**Steps:**
- Keep upstream Chatwoot files intact: `Client`, `SyncService`,
  `ChatwootHandler`, `/chatwoot/webhook`, and `/chatwoot/sync*`.
- Preserve the `3b87f4e` finding: `/chatwoot/webhook` bypasses basic auth, but
  outgoing webhook HMAC paths are unchanged; no HMAC client change is planned.
- Wire exactly 8 vars: `CHATWOOT_ENABLED`, `CHATWOOT_URL`, `CHATWOOT_API_TOKEN`,
  `CHATWOOT_ACCOUNT_ID`, `CHATWOOT_INBOX_ID`, `CHATWOOT_DEVICE_ID`,
  `CHATWOOT_IMPORT_MESSAGES`, `CHATWOOT_DAYS_LIMIT_IMPORT_MESSAGES`.
- In the same window, chatwoot-app stops relying on the legacy custom parser for
  duplicated gateway behavior and consumes the upstream chatwoot-aware path.
- Run `cd src && go test ./infrastructure/chatwoot ./infrastructure/whatsapp ./ui/rest`
  plus a paired-phone message -> Chatwoot conversation -> reply scenario.
**Checkpoint:** Staging lockstep deploy creates/finds Chatwoot contacts and
conversations, forwards message/reaction events, sends replies, and reports
`/chatwoot/sync*` progress.
**Rollback:** Highest-risk rollback. Lockstep means both repos are stuck if this
breaks after cutover. Roll back both artifacts together: GoWA Slice 3 image/config
and the previous chatwoot-app release. Rolling back only one repo is unsafe.
### Slice 5 - Webhook taxonomy and consumer compatibility
**Goal:** Ship the union webhook contract and verify gateway plus chatwoot-app
handle upstream event/payload expansion plus fork additions.
**Files touched:** `src/infrastructure/whatsapp/webhook_forward.go`;
`src/infrastructure/whatsapp/{event_message,event_message_handler,event_group,event_newsletter,auto_reply}.go`;
`docs/webhook-payload.md`; `readme.md`; `src/.env.example`;
`src/config/settings.go`.
**Steps:**
- Verify docs/events include upstream `chat_presence` (`c428afa`), `call.offer`
  (`5c193bc`), contact `phone_number` (`437df12`), `contacts_array` (`00ee65b`),
  CTWA referral keys (`fe7d2c7`), media `caption` (`306391e`), and fork
  `history_sync_complete`.
- Keep fork `chat_name` and `sender_name` for current consumers.
- Treat `is_from_me` as additive and `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` as
  deprecated/dead: present in `.env.example`, not an effective Go filter.
- Update chatwoot-app assumptions: suppress echoes using `is_from_me` or
  Chatwoot message type, not the dead outgoing knob.
- Run `rg -n "chat_presence|call.offer|history_sync_complete|is_from_me|WHATSAPP_WEBHOOK_INCLUDE_OUTGOING|chat_name|sender_name" src docs readme.md`
  and controlled phone actions for message, reaction, typing, call, media, and
  history sync.
**Checkpoint:** Every documented event is emitted or fixture-verified, HMAC is
unchanged, chatwoot-app ignores the dead outgoing knob, and no top-level payload
key is removed.
**Rollback:** Prefer a forward compatibility patch gating event names through
`WHATSAPP_WEBHOOK_EVENTS`. Resetting below Slice 4 re-enters lockstep rollback,
so only roll back to Slice 4 if the Chatwoot cutover is stable.
### Slice 6 - Release hardening and v8.5.0+1 cut
**Goal:** Package the reset+reapply result as `v8.5.0+1` with docs, Helm, Docker,
and release workflows aligned.
**Files touched:** `CHANGELOG.md`; `readme.md`; `docs/webhook-payload.md`;
`docs/chatwoot.md`; `src/.env.example`; `charts/gowa/{Chart.yaml,values.yaml,README.md}`;
`.github/workflows/{build-docker-image,release,chart-releaser,set-latest-tag}.yaml`.
**Steps:**
- Finalize `CHANGELOG.md` with `## [v8.5.0+1]`, `### Upstream Changes`, and
  `### Fork Changes`.
- Confirm release docs/workflows formalize `vX.Y.Z+N` and OCI tag rewrite.
- Confirm Helm exposes the 8 `CHATWOOT_*` vars, new upstream WhatsApp vars, and
  the deprecated outgoing knob without adding `CHATWOOT_IMPORT_CONTACTS`.
- Run `cd src && go test ./...`, `git diff --check`, `helm lint charts/gowa`,
  and the repo's release dry-run.
**Checkpoint:** `v8.5.0+1` image, release artifacts, and Helm chart are produced
from one commit after Slice 4 and Slice 5 checkpoints are recorded.
**Rollback:** Before tagging, reset to the latest slice checkpoint. After tagging,
publish `v8.5.0+2` or deprecate/yank per release policy; do not rewrite public
tags without an explicit human decision.
## 2. Slice ordering rationale

The order starts with the reset baseline because upstream `v8.5.0` is the new
foundation. Slice 0 restores packaging first so every later behavior is
deployable. Slice 1 moves next because BR phone normalization touches the widest
usecase surface and gives an early paired-phone routing signal. Slice 2 follows
because investigation proved LID dedup is not subsumed; it is an end-to-end
storage/history/webhook behavior. Slice 3 gathers remaining gateway behavior and
the fork-heavy overlap sweep before Chatwoot couples those files to Rails.
Slice 4 is late because lockstep cross-repo rollback is expensive. Slice 5 then
validates the public webhook contract with native Chatwoot live. Slice 6 cuts the
release only after behavior and compatibility checkpoints exist.
## 3. Open at P-time
### OQ3 file-boundary call

Pick `src/infrastructure/whatsapp/forward_history_sync.go` as a new fork file.
S confirms `forwardPayloadToConfiguredWebhooks(ctx, payload, eventName) error`
already exists with the same signature in fork and upstream, so this is only a
file-boundary choice. A new file isolates fork-only `history_sync_complete` from
upstream's large `webhook_forward.go` Chatwoot split and reduces future conflicts.
### OQ8+9 fork-heavier files

Use a bundled sweep, not standalone file slices. `device_manager.go` and
`usecase/{chat,group,message}.go` are not shippable by file; their behavior
belongs to Slice 1 phone routing, Slice 2 LID/history, and Slice 3 proxy/cache
device plumbing. Split only if Slice 3 inspection finds a concrete conflict.
### `is_from_me` and `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING`

`is_from_me` is the stable additive consumer-side signal for incoming/outgoing
payloads. `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` is deprecated/dead, not removed;
outgoing messages are effectively always forwarded. chatwoot-app must suppress
echoes from payload data (`is_from_me` or Chatwoot message type), not from the
gateway env knob.
## 4. Estimated effort

Slice 0: 0.5 person-day. Risk: wrong reset base or release tag shape. Mitigation:
SHA checks against R and a checkpoint tag.

Slice 1: 1.0 person-day. Risk: 45 usecase call sites and test movement.
Mitigation: keep S's `ValidateAndNormalizeJID` shape and bundle BR tests.

Slice 2: 1.0-1.5 person-days. Risk: storage/history race around LID context.
Mitigation: paired-phone full history sync with real duplicate LID chats.

Slice 3: 1.5-2.0 person-days. Risk: broad gateway behavior plus fork-heavy
overlap files. Mitigation: finish before Slice 4 and split only real conflicts.

Slice 4: 2.0 person-days plus chatwoot-app coordination. Highest risk: reset
plus lockstep cutover means both repos are stuck if it breaks. Mitigation:
staging lockstep deploy, no HMAC changes, exact 8-var env wiring, and prepared
rollback to GoWA Slice 3 plus previous chatwoot-app.

Slice 5: 1.0 person-day. Risk: consumer compatibility around event taxonomy and
forced outgoing forwarding. Mitigation: event-by-event phone actions and explicit
chatwoot-app handling of `is_from_me`.

Slice 6: 0.5-1.0 person-day. Risk: release metadata drift. Mitigation:
`go test ./...`, release dry-run, `helm lint`, and no tag until Slice 4/5 pass.

Total: roughly 7.5-9.0 person-days. Main risk concentrations are Slice 4's
lockstep rollback and Slice 3 if fork-heavy files hide behavior conflicts.
