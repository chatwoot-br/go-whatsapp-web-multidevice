# P — Plan (vertical slices, consolidated final)

QRSPI P stage. Final consolidation of `04-plan-claude.md` (244 lines, 8 slices, ~7.75d) and `04-plan-codex.md` (238 lines, 7 slices, 7.5-9.0d). Consolidation choices documented in `04-plan-consolidation.md`.

Strategy (locked under Q1=Reset+re-apply, Q2=Adopt+migrate chatwoot-app, Q3=Layer BR rules, Q4=All-in): hard-reset `main` to `upstream/main`, re-apply fork deltas as a small set of fresh commits on top. Slices below are the vertical grouping of those reapplies, ordered to (a) put the ship-stoppable lockstep cutover on the cleanest possible tree, (b) keep each slice's checkpoint independently diagnosable, (c) preserve the rollback anchor at `pre-upgrade-snapshot-2026-05-14` for every post-reset slice.

Default rollback for every post-reset slice: `git reset --hard pre-upgrade-snapshot-2026-05-14` + redeploy prior image. Per-slice rollback notes call out where stronger discipline applies (Slice 4 = cross-repo coordinated).

## 1. Slice list

### Slice 0 — Preflight

**Goal:** Snapshot fork state and create the upgrade branch. Establish the rollback anchor BEFORE any reset.

**Files touched:** None (tag + branch only).

**Steps:**
- `git fetch upstream --tags && git rev-parse upstream/main v8.5.0 main v8.1.2+1` — confirm R-stage SHAs (`17af98e`, `2e1798b`, `b4fd010`, `b4fd010`).
- `git tag -a pre-upgrade-snapshot-2026-05-14 main -m "pre v8.5 sync snapshot"` + `git push origin pre-upgrade-snapshot-2026-05-14`.
- `git switch -c upgrade/v8.5.0-sync main`.
- `cd src && go test ./... && go vet ./...` — capture pre-reset baseline green.

**Checkpoint:** Snapshot tag pushed; branch exists at `b4fd010`; `go test ./...` green pre-reset.

**Rollback:** Discard branch (`git branch -D upgrade/v8.5.0-sync`). Tag is immutable archaeology.

### Slice 1 — Reset + smoke + release rail

**Goal:** Land `upstream/main` as the new fork baseline, prove the upstream-only tree builds and pairs a phone, then re-apply the fork's deployment infrastructure (Helm chart, CI workflows, Docker tag rewrite, CHANGELOG skeleton). Smoke FIRST so a `go test` failure is diagnosable as "upstream regression" vs "release-rail YAML conflict."

**Files touched:**
- 134-file delta from `main..upstream/main` (lands via `git reset --hard`).
- Re-applied as a single post-reset commit: `charts/gowa/{Chart.yaml,values.yaml,templates/,README.md}`, `.github/workflows/{build-docker-image.yaml,release.yml,chart-releaser.yaml,set-latest-tag.yaml}`, `docker/golang.Dockerfile` (mailcap), `CHANGELOG.md` skeleton.

**Steps:**
- `git reset --hard upstream/main` on `upgrade/v8.5.0-sync`.
- `cd src && go mod tidy && go build ./... && go test ./...` — upstream tree green against locked deps. If RED here, **flag back to D** (upstream bug), don't paper over with fork reapply.
- Build Docker image locally; boot gateway against staging Postgres; QR-pair the staging phone (default fixture per Open §4); send + receive one round-trip message.
- Capture upstream-baseline session DB as portable fixture for later slices.
- **Then** re-apply release rail in one commit: Helm chart, four CI workflows, Dockerfile `mailcap` line, `## [v8.5.0+1] - (in progress) (Synced with upstream v8.5.0)` CHANGELOG header.
- `helm lint charts/gowa/` and a dry-run of `release.yml` (verify `vX.Y.Z+N` tag-shape regex accepts the planned tag).

**Checkpoint:** `go test ./...` green; gateway pairs phone; one round-trip on upstream-vanilla; Helm lint passes; CI workflow accepts `v8.5.0+1` shape on dry-run.

**Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. No state outside the branch has moved.

### Slice 2 — BR phone layer + caller sweep + tests

**Goal:** Re-apply BR 9-digit normalization as a thin layer on upstream's `pkg/utils/phone.go`. End-to-end testable: a BR mobile number reaching any usecase entry point normalizes to the canonical 12-digit form.

**Files touched:**
- NEW: `src/pkg/utils/phone_br.go`, `src/pkg/utils/phone_br_test.go`.
- MODIFIED: `src/pkg/utils/whatsapp.go` (drop fork's `ValidateAndNormalizeJID` — moved to `phone_br.go`; upstream's `ValidateJidWithLogin` retained per S).
- MODIFIED: ~40 caller sites in `src/usecase/{send.go, group.go, message.go, chat.go, user.go, newsletter.go}` — audit exact count at slice start via `rg -n 'utils\.ValidateAndNormalizeJID\(' src/usecase/` (Claude=39, Codex=45; neither cited grep, so audit fresh).
- MOVED: `src/infrastructure/whatsapp/jid_utils_test.go` BR fixtures + `pkg/utils/{whatsapp_test.go,general_test.go}` BR-specific cases folded into `phone_br_test.go` (D-review OQ10 bundle decision).

**Steps:**
- Write `phone_br.go` per S contract: `ValidateAndNormalizeJID(client *whatsmeow.Client, jid string) (types.JID, error)` wraps `utils.NormalizePhoneE164` + `utils.ParseJID` + `client.IsOnWhatsApp` under a 10s context timeout. Internal `normalizePhoneBR(phone string) string` stays unexported (Open §5).
- Run the rg audit to confirm caller count and inspect each call site for upstream signature drift on the call argument shape.
- Update each caller site: preserve `(client, jid) → (types.JID, error)` contract verbatim.
- Move BR test fixtures per OQ10.
- `cd src && go test ./pkg/utils/... ./usecase/...` green.

**Checkpoint:** `TestValidateAndNormalizeJID_BR9thDigit*` unit tests green; paired BR phone receives a send-API call with a 13-digit (`55669...`) number and the message lands without a "not on WhatsApp" error.

**Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. Mid-chain `git revert` of the caller sweep is not viable.

### Slice 3 — LID dedup + `history_sync_complete` dispatch

**Goal:** Re-apply post-history-sync LID dedup (OQ2 confirmed not subsumed) and plug fork's `history_sync_complete` event into upstream's restructured dispatcher.

**Files touched:**
- MODIFIED: `src/infrastructure/whatsapp/history_sync.go` — `deduplicateLIDChats` post-sync pass; debounced history-sync notification.
- MODIFIED: `src/infrastructure/whatsapp/jid_utils.go` — `NormalizeJIDFromLIDWithContext` (the no-context `NormalizeJIDFromLID` ↔ upstream `ResolveLIDToPhone` cleanup deferred to Slice 6 follow-up).
- MODIFIED: `src/infrastructure/chatstorage/sqlite_repository.go`, `src/infrastructure/chatstorage/device_repository.go`, `src/infrastructure/whatsapp/chatstorage_wrapper.go`, `src/domains/chatstorage/interfaces.go` — `MergeLIDChat`, `GetLIDChats` (per S signature block).
- NEW: `src/infrastructure/whatsapp/forward_history_sync.go` (Open §1 → new file).
- MODIFIED: `src/.env.example`, `src/config/settings.go` — extend `WHATSAPP_WEBHOOK_EVENTS` default to include `history_sync_complete`.
- MODIFIED: `docs/webhook-payload.md`, `readme.md` — add `history_sync_complete` to event table next to upstream's `chat_presence` + `call.offer`.

**Steps:**
- Re-add interface methods first (`interfaces.go`), then implementations (`sqlite_repository.go`, `device_repository.go`, `chatstorage_wrapper.go`); compile gate keeps the chain honest.
- Re-apply `deduplicateLIDChats` between `applyCachedPushNamesToChats` and the history-sync completion webhook in `history_sync.go` (insertion point in `docs/decisions/2026-01-18-fix-history-sync-lid-duplicate-chats.md`).
- Write `forward_history_sync.go`: single function calling `forwardPayloadToConfiguredWebhooks(ctx, payload, "history_sync_complete")` (S-locked signature, identical on both sides — purely code-organization).
- `cd src && go test ./infrastructure/whatsapp/... ./infrastructure/chatstorage/... ./domains/chatstorage/...`.

**Checkpoint:** Paired BR phone with at least one `@lid`-server contact completes history sync; assert (a) zero rows with `chat_jid LIKE '%@lid'` in `device_<id>.db.chats`, (b) exactly one `POST /webhook` with `event: history_sync_complete` at the test receiver.

**Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. Re-run Slices 1+2.

### Slice 4 — Chatwoot lockstep cutover (HIGHEST RISK)

**Goal:** Both repos flip in one window. Gateway activates upstream's native chatwoot integration; chatwoot-app's `Channel::Whatsapp::Provider` cuts over from the legacy custom-JSON webhook shape to upstream's chatwoot-aware push payload. **Includes `is_from_me`-based echo suppression on consumer side** (S correction + Open §3) — behavior, not docs.

**Files touched (gateway, already in tree via Slice 1 reset):**
- `src/infrastructure/chatwoot/{client,sync,sync_types,types,sync_test}.go` (upstream — verify intact after Slice 1).
- `src/ui/rest/chatwoot.go`, `src/cmd/rest.go` (verify `/chatwoot/webhook` registered BEFORE basic-auth middleware per `3b87f4e`).
- `src/infrastructure/whatsapp/webhook_forward.go` — upstream's `forwardToChatwoot` + `forwardToWebhooks` already in place; fork's `forward_history_sync.go` (Slice 3) continues dispatching alongside.

**Files touched (this slice's edits):**
- MODIFIED: `src/.env.example`, `src/config/settings.go`, `src/cmd/root.go`, `charts/gowa/values.yaml`, `charts/gowa/templates/configmap.yaml` — wire 8 `CHATWOOT_*` env vars (per S correction: count is 8 not 9; `CHATWOOT_IMPORT_CONTACTS` was added then removed inside `3b87f4e`).
- MODIFIED: `docs/webhook-payload.md`, `readme.md` — chatwoot section + deprecation note on `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` (S correction: dead knob still in env.example).

**Files touched (chatwoot-app, separate repo, separate PR):**
- `Channel::Whatsapp::Provider#process_messages` — swap legacy fork-shape parsing for upstream chatwoot push payload consumption at `/webhooks/whatsapp/:hmac_token`.
- Retire `format_status_message` parser branches keyed on fork's `chat_name`, `sender_name`, `history_sync_complete`.
- **`is_from_me`-based echo suppression**: any consumer-side filter that currently relies on `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` switches to checking `is_from_me` on the payload. The env knob is dead in upstream; outgoing messages always reach the webhook. Branch on `is_from_me` to skip outgoing echoes.

**Steps:**
- Confirm `3b87f4e` route-order intact: `/chatwoot/webhook` excluded from basic-auth, `/chatwoot/sync*` behind basic-auth.
- Populate gateway env (eight knobs: `CHATWOOT_ENABLED`, `CHATWOOT_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`, `CHATWOOT_INBOX_ID`, `CHATWOOT_DEVICE_ID`, `CHATWOOT_IMPORT_MESSAGES`, `CHATWOOT_DAYS_LIMIT_IMPORT_MESSAGES`) in staging.
- Helm values bumped to expose the eight knobs.
- chatwoot-app PR pre-prepared and reviewed BEFORE this slice's cutover window. Both PRs merge-ready when window opens.
- Staging dry-run: deploy both repos together; round-trip the five scenarios from the checkpoint below.
- **Cross-repo rollback rehearsal in staging** (deploy → roll back both repos → redeploy) BEFORE prod cutover. Non-negotiable.
- Prod cutover: deploy gateway + chatwoot-app together in one maintenance window.

**Checkpoint:** Five paired-phone scenarios green on prod:
1. Inbound text message lands in chatwoot inbox as a new conversation.
2. Chatwoot agent reply delivers to the paired phone.
3. Inbound `message.reaction` updates the chatwoot message.
4. Outgoing-from-phone (non-chatwoot-originated) message lands in chatwoot with `is_from_me=true`; consumer echo-suppression filter respects it.
5. Echo-loop guard (`MarkMessageAsSent` / `IsMessageSentByUs` per S `client.go` signature) prevents double-posting.

**Rollback:** **Cross-repo coordinated.** `git reset --hard pre-upgrade-snapshot-2026-05-14` + redeploy prior gateway image + revert chatwoot-app PR + redeploy prior chatwoot-app image, all in the same window. Pre-stage both rollback images; on-call comms before cutover; staging rollback rehearsal mandatory.

### Slice 5 — Webhook taxonomy + env audit

**Goal:** Document upstream's expanded event taxonomy and the field-level changes that landed via Slice 1 (reset) and Slice 4 (chatwoot integration). Verify each event type triggers correctly via paired-phone actions. **Documentation-side of Open §3** (is_from_me field + INCLUDE_OUTGOING dead-knob note).

**Files touched:**
- MODIFIED: `docs/webhook-payload.md`, `readme.md` — final pass on event taxonomy table; deprecated-note under `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` ("knob is dead config; outgoing webhooks always forwarded"); document `is_from_me` field under `### Common Payload Fields`.
- MODIFIED: `charts/gowa/values.yaml`, `charts/gowa/README.md` — Helm surface for upstream's new WhatsApp env vars (`WHATSAPP_AUTO_REJECT_CALL`, `WHATSAPP_PRESENCE_ON_CONNECT` if present).

**Steps:**
- Verify `WHATSAPP_WEBHOOK_EVENTS` default list includes upstream's `chat_presence`, `call.offer`, fork's `history_sync_complete`.
- Paired-phone trigger sequence:
  - Typing on staging phone → `chat_presence` arrives at test receiver.
  - Incoming call to staging phone → `call.offer` arrives.
  - Full history sync cycle (re-verify from Slice 3) → `history_sync_complete` arrives.
  - Outgoing message from phone → `is_from_me=true` in payload.
  - Contact share → `contacts_array` payload shape (per `00ee65b`).
  - Media with caption → caption field present (per `306391e`).
- `rg -n 'chat_presence|call\.offer|history_sync_complete|is_from_me|chat_name|sender_name|contacts_array' src/ docs/ readme.md` — final taxonomy reach audit.

**Checkpoint:** Test webhook receiver records all documented event types with paired-phone-triggered payloads; doc pages match S's webhook contract table.

**Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. Docs-only slice.

### Slice 6 — Fork-only delta sweep + OQ8 device_manager.go

**Goal:** Re-apply remaining fork-only deltas (audio/PTT, proxy, info-cache, S3 ext fix, APP_BASE_PATH, jid-helper cleanup). Reconcile `device_manager.go` (fork-heavier 3 vs 2). **Inspect-first, split-only-on-conflict rule** (Open §2). Mandatory chatwoot regression check at slice end (Codex's concern: fork-side `device_manager.go` edits can shadow `DeviceInstance` assumptions chatwoot depends on).

**Files touched (sweep, one commit each):**
- Audio/PTT (v8.1.0+1): `src/usecase/send.go` PTT path, ffmpeg dependency, audio waveform + MIME/duration helpers, `docker/golang.Dockerfile` if not absorbed by Slice 1's mailcap commit.
- SOCKS5/HTTP/HTTPS proxy (v8.1.0+3): `src/config/settings.go`, `src/infrastructure/whatsapp/client_lifecycle.go` or post-reset equivalent, UI proxy-IP display in `src/views/`.
- Short-term info-request cache (v8.1.0+2): `src/infrastructure/whatsapp/info_cache.go` + `src/pkg/cache/cache.go`.
- S3 image-extension fix (v8.1.0+5): `src/pkg/utils/general.go` Content-Type-derived extension.
- `APP_BASE_PATH` middleware fix (v8.1.0+1): `src/cmd/rest.go` if upstream's restructure didn't supersede it.
- jid-helper cleanup (OQ2 follow-up): drop fork's no-context `NormalizeJIDFromLID` (byte-identical to upstream's `ResolveLIDToPhone`); route the fork-unique `NormalizeJIDFromLIDWithContext` through upstream's helper as its base.

**Files touched (OQ8+9 inspect-first):**
- `src/infrastructure/whatsapp/device_manager.go` — fork 3 vs upstream 2. Run `git log v8.1.2..upstream/main -- src/infrastructure/whatsapp/device_manager.go` at slice start. If upstream's edits don't overlap fork-edited functions: bundle into sweep. If they do: own commit with three-way diff. Budget +0.5d if overlap.

**Steps:**
- For each sweep item, `git show <fork-commit-SHA>` from `pre-upgrade-snapshot-2026-05-14`, hand-port to current tree, run scoped tests.
- Run the OQ8 inspection check on `device_manager.go` (and any other R-flagged fork-heavier file not absorbed by earlier slices).
- `cd src && go test ./...` after each commit.
- **Mandatory chatwoot regression check at slice end**: re-run the five paired-phone chatwoot scenarios from Slice 4. Fork-side `device_manager.go` edits CAN shadow chatwoot's `DeviceInstance` assumptions; this checkpoint catches it before release.

**Checkpoint:** Full `go test ./...` green; paired-phone audio (OGG Opus PTT) sends and plays; proxy-IP renders in device-card UI; S3 image upload produces Content-Type-matching extension; `helm template charts/gowa/ | kubectl apply --dry-run=client` passes; **all five chatwoot scenarios from Slice 4 still green**.

**Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. Sweep commits don't compose under partial revert.

### Slice 7 — Release: tag, CHANGELOG, CI formalization

**Goal:** Cut `v8.5.0+1`; publish image and Helm chart; formalize the `vX.Y.Z+N` convention in CI so future syncs can't drift (Q4 enhancement).

**Files touched:**
- MODIFIED: `CHANGELOG.md` — finalize `## [v8.5.0+1] - 2026-MM-DD (Synced with upstream v8.5.0)` with `### Upstream Changes` (88 commits summarized by bucket from R) + `### Fork Changes` (per-slice fork-delta commits from Slices 2-6).
- MODIFIED: `src/config/settings.go` — `AppVersion = "v8.5.0+1"` bump.
- MODIFIED: `charts/gowa/Chart.yaml` — `version` + `appVersion` bump.
- MODIFIED: `.github/workflows/release.yml` — formalize `vX.Y.Z+N` tag-shape gate (regex on tag trigger; reject malformed shapes).
- VERIFIED: `.github/workflows/build-docker-image.yaml:24` `+`→`-` rewrite still active.

**Steps:**
- Author CHANGELOG entry; PR for review.
- `git tag v8.5.0+1 && git push origin v8.5.0+1` after merge.
- GoReleaser workflow runs on tag push; verify Docker image lands as `:v8.5.0-1` (post-rewrite).
- `chart-releaser.yaml` publishes Helm chart on `v*` tag.
- chatwoot-app deployment pin updated to `v8.5.0-1` image; final deploy.

**Checkpoint:** `docker pull ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v8.5.0-1` succeeds; `helm pull` of new chart succeeds; chatwoot-app prod is on the new image.

**Rollback:** Tags are immutable. To rollback the release: cut `v8.5.0+2` reverting the broken slice, or redeploy `v8.1.2-1` from registry. Don't `git tag -d` a pushed tag.

## 2. Slice ordering rationale

Slices 0+1 are non-negotiable first: without preflight + reset + smoke + release-rail, every later slice's breakage is ambiguous between "reset broke it," "fork reapply broke it," and "deploy infrastructure broke it." Bundling release rail into Slice 1 (with smoke FIRST within the slice) gives diagnostic clarity AND makes CI workflows functional for every later slice's PR review.

Slice 2 (BR phone) goes next because it touches ~40 caller sites across 6 usecase files; deferring means doubling sweep effort when Slice 6 revisits the same surface area. Slice 3 (LID + history_sync_complete) follows because its chatstorage interface changes ripple into the wrapper, and any slice that consumes `IChatStorageRepository` (notably upstream's `SyncService` activated in Slice 4) compiles against the fork-extended interface. Slice 3's storage interface is the soft contract Slice 4 depends on.

Slice 4 (chatwoot lockstep) sits mid-chain on a clean tree: depends on Slice 3's storage interface and Slice 2's phone normalization (chatwoot syncs BR numbers); gates Slice 5's webhook audit (the `is_from_me` behavior change affects the consumer). Pushing chatwoot later means longer chatwoot-app PR aging; pushing it earlier means activating chatwoot on a tree that doesn't yet handle BR phones or LID dedup. Mid-chain is the constrained optimum.

Slice 5 (taxonomy + env audit) is mostly docs + verification: it follows Slice 4 because `is_from_me` documentation is a Slice 4 behavioral consequence. Slice 6 (fork sweep + `device_manager.go`) is late because its items are low-risk fork-only reapplies that benefit from the BR/LID/chatwoot path already being stable AND because the mandatory chatwoot-regression check at Slice 6's end catches any fork-edit shadowing of `DeviceInstance` assumptions before release. Slice 7 (release) closes.

Vertical-slicing invariant: each slice ends with an end-to-end paired-phone scenario green. Slices 1, 2, 3, and 5 are independently shippable as RC images (`v8.5.0+1-rc1`, `-rc2`, etc.) if mid-chain risk emerges. Slice 4 is the ship-or-rollback gate.

## 3. Open at P-time

1. **OQ3 file boundary → Option A: new file `src/infrastructure/whatsapp/forward_history_sync.go`.** Rationale: the reapply diff is one new file rather than a hunk inside upstream's restructured 484-line `webhook_forward.go`; easier to audit, easier to submit upstream as a future PR. S confirmed `forwardPayloadToConfiguredWebhooks(ctx, payload, eventName) error` exists with identical signature on both sides — purely code-organization. Both drafts (Claude + Codex) agreed on this call.

2. **OQ8+9 fork-heavier files → bundle-by-default in Slice 6 with mandatory inspect-first step.** `device_manager.go` is the only genuinely orphan fork-heavier file (Slice 2 absorbs the usecase callers; Slice 3 absorbs `history_sync.go`). Run `git log v8.1.2..upstream/main -- <file>` at Slice 6 start; if upstream's edits don't overlap fork-edited functions, bundle into sweep. Split only on overlap. The Slice 6 chatwoot regression check is the safety net: if `device_manager.go` reapply shadows `DeviceInstance` assumptions, the chatwoot round-trip test fails the checkpoint. (Codex's rule; Claude's flagging risk addressed by the regression check.)

3. **`is_from_me` + `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` split across slices.** *Behavior* change (chatwoot-app's echo-suppression filter switches from env knob to payload field) lands in Slice 4 — the lockstep window is where the consumer logic flips, and lockstep ruled out dual-listen. *Documentation* change (`docs/webhook-payload.md` deprecation note, `### Common Payload Fields` update) lands in Slice 5 — part of the taxonomy audit. (Claude's call on behavior; Codex's framing for docs; resolved by splitting.)

4. **Slice 1 paired-phone fixture → re-pair staging phone.** Three options were considered: (a) throwaway burner, (b) re-pair existing staging phone, (c) checkpoint-restore session DB into post-reset build. Default (b) — 60-second action, avoids fixture rot, no schema compatibility unknowns. (From Claude draft; no objection from Codex.)

5. **`normalizePhoneBR` visibility → unexported.** BR fixtures live in the same package (`phone_br_test.go`); internal-test access is sufficient. Exporting just to satisfy a future external consumer is YAGNI. (From S Open list; resolved.)

## 4. Estimated effort

| Slice | Days | Risk notes |
|---|---:|---|
| 0 — Preflight | 0.25 | Trivial. Tag + branch + baseline-green check. |
| 1 — Reset + smoke + release rail | 1.0 | Two sub-checkpoints in one slice. If `go test ./...` fails on upstream-vanilla post-reset, flag back to D (upstream bug) — don't paper over with fork reapply. Helm + CI reapply uses existing fork heredoc; low conflict risk. |
| 2 — BR phone + ~40 callers + tests | 1.5 | Caller count is the size driver. Per-site swap is mechanical but each needs an eyeball pass for upstream signature drift on the call argument. |
| 3 — LID dedup + history_sync_complete | 1.0 | Interface re-add ripples through wrapper + repository; OQ3 file-boundary locked (new file). |
| 4 — Chatwoot lockstep cutover | 2.0 | **Highest risk.** Cross-repo PR coordination + staging dry-run + prod cutover window. Assumes chatwoot-app PR is pre-prepared and reviewed in parallel. If chatwoot-app PR slips, lockstep window slips. **Staging rollback rehearsal mandatory.** |
| 5 — Webhook taxonomy + env audit | 0.5 | Mostly docs + paired-phone trigger verification. |
| 6 — Fork sweep + OQ8 device_manager.go | 1.5 | Five sweep items + one inspect-first file. `device_manager.go` three-way is the size driver if overlap surfaces; budget **+0.5d** if upstream's 2 commits hit fork-edited functions. **Mandatory chatwoot regression check at slice end.** |
| 7 — Release + CI formalization | 0.5 | CHANGELOG drafting + tag push + workflow regex addition. |
| **Total** | **8.25** | Single-engineer serial (range: 8.25-8.75 depending on Slice 6 inspection outcome). |

### Risk concentrations

- **Slice 4 is the ship-stoppable gate.** Lockstep cutover means chatwoot-app must be PR-ready, reviewed, and merge-able in the same window as the gateway PR. Staging rollback rehearsal mandatory; on-call comms before prod cutover; both rollback images pre-staged.

- **Reset+reapply rollback granularity.** Every post-reset slice's rollback is "branch reset to `pre-upgrade-snapshot-2026-05-14` and redeploy prior image." Mid-chain `git revert` of Slice N forces replay of Slices N+1..M. Mitigation: keep slice commits small and atomic; never bundle two slices into one commit.

- **OQ8 `device_manager.go` deferred inspection.** If upstream's 2 commits hit functions fork-edited, Slice 6's bundled approach balloons to per-file own-commit with three-way conflict resolution. Surface day-of as a Slice 6 sub-risk; budget +0.5d.

- **Fork-edit shadowing of chatwoot `DeviceInstance` assumptions.** Fork-side `device_manager.go` reapply (Slice 6) lands AFTER chatwoot integration goes live (Slice 4). The Slice 6 mandatory chatwoot-regression check is the safety net; without it, fork-delta could silently break chatwoot in prod between Slice 6 commit and Slice 7 release.

- **chatwoot-app PR aging.** Slice 4 sits mid-chain (~4-5 days into the upgrade). The chatwoot-app PR must be drafted around Slice 2 to be ready for parallel review by Slice 4. Aging beyond ~10 days risks the Rails-side context drifting from gateway expectations; sync points between teams advised.
