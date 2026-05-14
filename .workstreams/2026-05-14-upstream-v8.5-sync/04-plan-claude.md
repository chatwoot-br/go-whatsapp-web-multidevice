# P — Plan (vertical slices) — Claude draft

QRSPI P stage. Produced from S (`03-structure.md`) under the locked Q answers
(Q1=Reset+re-apply, Q2=Adopt+migrate chatwoot-app, Q3=Layer BR rules,
Q4=All-in). The speculative `04-plan.md` was reshaped — under reset+re-apply,
whatsmeow/CTWA/GIF/ghost-mentions/document-thumbnails ride in for free with
the hard reset; slices are framed around what's actually *done* (the reset and
the fork-delta reapply chunks), not around upstream features that arrive
gratis.

Each slice is end-to-end and checkpointable. Default rollback for **every**
post-reset slice is "branch reset to `pre-upgrade-snapshot-2026-05-14` and
re-deploy prior image" — the reapply chain doesn't tolerate a mid-chain
`git revert`. Slice rollbacks below name when something stronger is needed.

## Slice 0 — Preflight

- **Goal:** Snapshot fork state; create the upgrade branch; lock the baseline-green gate.
- **Files touched:** none (tag + branch only).
- **Steps:**
  - `git fetch upstream` → confirm `upstream/main` HEAD is `17af98e` (v8.5.0+5 per R).
  - `git tag pre-upgrade-snapshot-2026-05-14 main` and `git push origin tag pre-upgrade-snapshot-2026-05-14`.
  - `git checkout -b upgrade/v8.5.0-sync` from current `main` (`b4fd010`).
  - `cd src && go test ./... && go vet ./...` to capture baseline green.
- **Checkpoint:** Snapshot tag pushed; branch exists at `b4fd010`; `go test ./...` green pre-reset.
- **Rollback:** Discard branch (`git branch -D upgrade/v8.5.0-sync`). Snapshot tag is immutable archaeology.

## Slice 1 — Reset to upstream + baseline smoke

- **Goal:** Land `upstream/main` as the new fork baseline and prove the upstream-only gateway boots cleanly in fork CI before any fork delta re-applies. Surfaces upstream regressions without entangling them with reapply diffs.
- **Files touched:** every file in `main..upstream/main` (134 files, +6841/-8830 per R) — landed in one hard reset.
- **Steps:**
  - `git reset --hard upstream/main` on `upgrade/v8.5.0-sync`.
  - `cd src && go mod tidy && go build ./...` to confirm the upstream tree builds clean against the locked deps.
  - `cd src && go test ./...` — all upstream tests (incl. the new `chatwoot_forward_test.go`, `sync_test.go`) green.
  - Build Docker image locally; boot gateway against staging Postgres; QR-pair the dedicated test phone; send + receive one message in each direction.
  - Capture upstream-baseline session DB as a portable fixture for later slices (rebuilds avoid re-pairing).
- **Checkpoint:** `go test ./...` green; gateway pairs phone; one round-trip message confirmed against paired phone over upstream-vanilla build.
- **Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. No state outside the branch has moved yet — true zero-cost rollback.

## Slice 2 — BR phone layer (`phone_br.go` + 39 callers + tests)

- **Goal:** Re-apply BR 9-digit normalization as a thin layer on upstream's `pkg/utils/phone.go`. End-to-end testable: a BR number reaching any usecase entry point normalizes correctly.
- **Files touched:**
  - NEW: `src/pkg/utils/phone_br.go`, `src/pkg/utils/phone_br_test.go`.
  - MODIFIED: `src/pkg/utils/whatsapp.go` (drop fork's `ValidateAndNormalizeJID`; upstream's `ValidateJidWithLogin` retained per S).
  - MODIFIED (caller routing, 39 sites per local `grep`): `src/usecase/{send.go(13), group.go(12), message.go(7), chat.go(3), user.go(3), newsletter.go(1)}`.
  - MOVED/MERGED tests: `src/infrastructure/whatsapp/jid_utils_test.go` BR fixtures folded into `phone_br_test.go`; BR-specific cases from `pkg/utils/whatsapp_test.go` rebased on the layered entry point (per OQ10 bundle decision).
- **Steps:**
  - Write `phone_br.go` per S contract: `ValidateAndNormalizeJID(client, jid) (types.JID, error)` wraps `utils.NormalizePhoneE164` + `utils.ParseJID` + `client.IsOnWhatsApp` under a 10s context timeout. Internal `normalizePhoneBR(phone string) string` stays unexported (P-time call from S Open).
  - Sweep callers: `rg -n 'utils\.ValidateAndNormalizeJID\(' src/usecase/` to verify the 39 hits exist on the upstream baseline post-Slice-1 (some upstream signatures changed shape, so audit the call site before swapping import).
  - Move `jid_utils_test.go` BR fixtures into `phone_br_test.go`. Drop `ValidateJidWithLogin` callers in the same patch only where they were the fork's deprecated path; non-BR upstream callers stay on `ValidateJidWithLogin`.
  - `cd src && go test ./pkg/utils/... ./usecase/...` green.
- **Checkpoint:** Unit tests `TestValidateAndNormalizeJID_BR9thDigit*` (in `phone_br_test.go`) green; paired BR phone receives a send-API call with a 13-digit (`55669...`) number and the message lands without a "not on WhatsApp" error.
- **Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. The 39-caller sweep can't be cleanly reverted mid-chain — only nuke-and-redo.

## Slice 3 — LID dedup + `history_sync_complete` dispatch

- **Goal:** Re-apply post-history-sync LID dedup (OQ2 confirmed not subsumed) and plug fork's `history_sync_complete` event into upstream's restructured dispatcher. End-to-end: a fresh-paired account completes history sync without `@lid`-server chat duplicates and emits one debounced `history_sync_complete` webhook.
- **Files touched:**
  - MODIFIED: `src/infrastructure/whatsapp/history_sync.go` (re-apply `deduplicateLIDChats`, `forwardHistorySyncCompleteToWebhook`, debounce logic).
  - MODIFIED: `src/infrastructure/whatsapp/jid_utils.go` (re-apply `NormalizeJIDFromLIDWithContext`; the no-context `NormalizeJIDFromLID` overlaps byte-identically with upstream's `ResolveLIDToPhone` — leave the cleanup to Slice 6 follow-up sub-task).
  - MODIFIED: `src/infrastructure/chatstorage/sqlite_repository.go`, `src/infrastructure/chatstorage/device_repository.go`, `src/domains/chatstorage/interfaces.go` — re-add `MergeLIDChat` / `GetLIDChats` per S signature block.
  - NEW: `src/infrastructure/whatsapp/forward_history_sync.go` (OQ3 default — see "Open at P-time" §1).
  - MODIFIED: `src/.env.example`, `src/config/settings.go` — re-extend `WHATSAPP_WEBHOOK_EVENTS` default list to include `history_sync_complete`.
  - MODIFIED: `docs/webhook-payload.md` — re-add `history_sync_complete` to the "Available Webhook Events" table (next to upstream's `chat_presence` + `call.offer`).
- **Steps:**
  - Re-add interface methods first (`interfaces.go`), then their implementations (`sqlite_repository.go`, `device_repository.go`); compile gate keeps the wrapper honest.
  - Re-apply `deduplicateLIDChats` to run between `applyCachedPushNamesToChats` and `forwardHistorySyncCompleteToWebhook` in `history_sync.go` (insertion point documented in `docs/decisions/2026-01-18-fix-history-sync-lid-duplicate-chats.md`).
  - Write `forward_history_sync.go` with a single function calling `forwardPayloadToConfiguredWebhooks(ctx, payload, "history_sync_complete")` (S-locked signature).
  - `cd src && go test ./infrastructure/whatsapp/... ./infrastructure/chatstorage/...`.
- **Checkpoint:** Paired-phone scenario: pair a fresh BR account that has at least one `@lid`-server contact; complete history sync; assert (a) no rows with `chat_jid LIKE '%@lid'` survive in `device_<id>.db.chats`, and (b) exactly one `POST /webhook` with `event: history_sync_complete` arrives at the test webhook receiver.
- **Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. Re-establish Slice 2 by re-running Slice 2 steps.

## Slice 4 — Chatwoot lockstep cutover (HIGHEST RISK)

- **Goal:** Both repos flip in one window. Gateway adopts upstream's native chatwoot integration; `chatwoot-app` Rails consumer cuts over from the legacy custom-JSON webhook shape to upstream's `/chatwoot/forward` push payload. After this slice, chatwoot-app's `Channel::Whatsapp::Provider` consumes upstream's stable API.
- **Files touched (gateway):**
  - All upstream-native chatwoot files arrived via Slice 1 reset: `src/infrastructure/chatwoot/{client,sync,sync_types,types,sync_test}.go`, `src/ui/rest/chatwoot.go`, `src/infrastructure/whatsapp/{webhook_forward.go,chatwoot_forward_test.go}`.
  - MODIFIED: `src/cmd/rest.go` — chatwoot routes registered; `/chatwoot/webhook` excluded from basic-auth per `3b87f4e` (already arrived via reset — verify ordering).
  - MODIFIED: `src/config/settings.go`, `src/cmd/root.go`, `src/.env.example`, `readme.md`, `docs/webhook-payload.md`, `charts/gowa/values.yaml` — eight (not nine, per S correction) new `CHATWOOT_*` env vars surfaced.
  - **Fork plug-in retained from Slice 3** — `forward_history_sync.go` continues to dispatch the fork's bespoke event alongside upstream's chatwoot push path; the two are independent dispatchers reading the same payload map.
- **Files touched (chatwoot-app, separate repo, separate PR):**
  - `Channel::Whatsapp::Provider`'s `process_messages` switched from parsing the legacy fork-shape (`chat_name`, `sender_name`, `history_sync_complete` branches) to consuming upstream's chatwoot push payload at the existing `/webhooks/whatsapp/:hmac_token` endpoint.
  - Retire bespoke `format_status_message` parser branches.
  - **Behavior-only deprecation handling** for `is_from_me` + `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` (D-correction § + S-flag): the env knob is dead-code-but-still-documented in upstream `.env.example`; behavior is "outgoing messages always forwarded". Any consumer code branching on the missing flag, or filtering on `is_from_me`, must be audited. Action: add an `is_from_me`-aware filter in `Channel::Whatsapp::Provider` if the consumer needs to skip outgoing echoes; do *not* rely on the env knob.
- **Steps:**
  - Confirm `/chatwoot/webhook` is registered before basic-auth middleware in `src/cmd/rest.go` (sanity check — upstream `3b87f4e` already did this, but the reset reapply order matters).
  - Populate gateway env (`CHATWOOT_ENABLED`, `CHATWOOT_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`, `CHATWOOT_INBOX_ID`, `CHATWOOT_DEVICE_ID`, `CHATWOOT_IMPORT_MESSAGES`, `CHATWOOT_DAYS_LIMIT_IMPORT_MESSAGES`) in staging.
  - Helm values reflect the eight knobs; bump `charts/gowa/values.yaml` and `charts/gowa/templates/configmap.yaml` (or equivalent) in the same commit.
  - chatwoot-app PR pre-prepared and reviewed *before* this slice's cutover window; merge gates green on both PRs.
  - Staging dry-run: deploy both repos to staging; round-trip `inbound message → gateway → chatwoot inbox`; round-trip `chatwoot agent reply → gateway → WhatsApp`; verify `message.reaction` flows; verify the `is_from_me` behavior with a paired-phone outgoing message that originates outside chatwoot (should still arrive at chatwoot, possibly muted by the new filter).
  - Prod cutover: deploy gateway + chatwoot-app together in one maintenance window; flip DNS/feature flag atomically.
- **Checkpoint:** Five paired-phone scenarios green on prod:
  1. Inbound text message lands in chatwoot inbox as a new conversation.
  2. Chatwoot agent reply delivers to the paired phone.
  3. Inbound `message.reaction` updates the chatwoot message.
  4. Outgoing-from-phone (non-chatwoot-originated) message is recorded in chatwoot with `is_from_me=true` and respects the new filter.
  5. Existing-conversation echo-loop guard (`MarkMessageAsSent` / `IsMessageSentByUs` per S `client.go` signature) prevents double-posting.
- **Rollback:** **Cross-repo coordinated rollback required.** `git reset --hard pre-upgrade-snapshot-2026-05-14` + redeploy prior gateway image **and** revert chatwoot-app PR + redeploy prior chatwoot-app image, in the same window. Mitigation: pre-stage both rollback images; comms window with on-call before cutover; staging dry-run end-to-end before prod flip.

## Slice 5 — Webhook taxonomy + env audit

- **Goal:** Surface upstream's new event types (`chat_presence`, `call.offer`) to downstream consumers, finalize the env-var inventory (8 `CHATWOOT_*` + `WHATSAPP_AUTO_REJECT_CALL` + `WHATSAPP_PRESENCE_ON_CONNECT` + fork-extended `WHATSAPP_WEBHOOK_EVENTS`), and lock the deprecated-but-present `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` knob as dead config (per D-correction §).
- **Files touched:**
  - MODIFIED: `src/.env.example` (already in via reset — verify `history_sync_complete` is in the fork-extended `WHATSAPP_WEBHOOK_EVENTS` default).
  - MODIFIED: `docs/webhook-payload.md`, `readme.md` — final pass on the taxonomy table (upstream's `chat_presence` + `call.offer` + fork's `history_sync_complete`); add a note under `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` that the knob is deprecated and behavior is forced-on.
  - MODIFIED: `charts/gowa/values.yaml`, `charts/gowa/README.md` — Helm surface for the new env vars.
- **Steps:**
  - Trigger each new event via a paired-phone action: typing indicator (`chat_presence`); incoming call to the paired number (`call.offer`); reset cycle (`history_sync_complete`).
  - Verify the test webhook receiver logs each event with the expected payload shape per S "Webhook contract surface" table.
  - Document the `is_from_me` behavior change in `docs/webhook-payload.md` `### Common Payload Fields`.
- **Checkpoint:** Test webhook receiver records all three new event types with paired-phone-triggered payloads matching S's contract table.
- **Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. Docs-only slice; rolling back loses no behavior.

## Slice 6 — Fork sweep + OQ8 fork-heavier files

- **Goal:** Re-apply remaining fork deltas (audio/PTT, proxy, info-cache, S3 ext fix, Docker mailcap) in one sweep commit per deltum; give contentious fork-heavier overlap files their own commits.
- **Files touched (sweep, one commit each):**
  - Audio/PTT: `src/usecase/send.go` PTT path + ffmpeg dependency + Dockerfile (v8.1.0+1).
  - SOCKS5/HTTP/HTTPS proxy: `src/config/settings.go`, `src/infrastructure/whatsapp/client_lifecycle.go` (or wherever post-reset), UI proxy-IP display in `src/views/` (v8.1.0+3).
  - Short-term info-request cache: `src/infrastructure/whatsapp/info_cache.go` (already present in fork tree — re-apply on top of upstream's `device_manager.go` baseline).
  - S3 image-extension fix: `src/pkg/utils/general.go` Content-Type-derived extension (v8.1.0+5).
  - Docker `mailcap`: `docker/golang.Dockerfile` (v8.1.2+1).
  - Helm chart preserved as-is: `charts/gowa/*` (untouched by reset; verify still present after Slice 1).
  - CI workflows preserved/re-applied: `.github/workflows/build-docker-image.yaml` (`+`→`-` rewrite at :24), `.github/workflows/chart-releaser.yaml`, `.github/workflows/set-latest-tag.yaml`, `.github/workflows/release.yml` (fork-heavier — see OQ8 call below).
- **Files touched (own-commit per OQ8+9 call):**
  - `src/infrastructure/whatsapp/device_manager.go` — fork-heavier (3 vs 2). Its own commit; reapply the fork-side cache/state hooks; audit against upstream's 2 commits per `git log v8.1.2..upstream/main -- src/infrastructure/whatsapp/device_manager.go` (run at slice start).
  - `src/.github/workflows/release.yml` — fork-heavier (4 vs 2). Its own commit; reapply Docker tag rewrite + GoReleaser glue; audit conflicts.
- **Steps:**
  - For each sweep item, `git show <fork-commit-SHA>` against the pre-upgrade-snapshot tag, hand-port to the upstream baseline, run scoped tests.
  - For each own-commit file, three-way diff vs `upstream/main` and `pre-upgrade-snapshot-2026-05-14`, hand-resolve, scope-test.
  - `cd src && go test ./...` after each commit.
- **Checkpoint:** Full `go test ./...` green; paired-phone audio (OGG Opus PTT) sends and plays; proxy-IP renders in device-card UI; S3 image upload produces extension matching Content-Type; Helm install via `helm template charts/gowa/ | kubectl apply --dry-run=client` passes.
- **Rollback:** `git reset --hard pre-upgrade-snapshot-2026-05-14`. Sweep commits don't compose cleanly under partial revert.

## Slice 7 — Release: tag, CHANGELOG, CI formalization

- **Goal:** Cut `v8.5.0+1`, publish image, ship Helm chart bump; **formalize the `vX.Y.Z+N` convention in CI** (OQ4 sub-task) so future syncs can't drift.
- **Files touched:**
  - `CHANGELOG.md` — `## [v8.5.0+1] - 2026-05-14 (Synced with upstream v8.5.0)` with `### Upstream Changes` (88 commits, summarized) + `### Fork Changes` (the slices 2-6 fork-delta commits).
  - `src/config/settings.go` — `AppVersion = "v8.5.0+1"` bump.
  - `charts/gowa/Chart.yaml` — `version` + `appVersion` bump.
  - `.github/workflows/release.yml` — formalize the `vX.Y.Z+N` tag-shape gate (regex-match the tag trigger; reject malformed shapes).
  - `.github/workflows/build-docker-image.yaml` — confirm `:24` `+`→`-` rewrite still active; lock the tag-shape regex into the workflow.
- **Steps:**
  - Author CHANGELOG entry; PR for review.
  - `git tag v8.5.0+1` and `git push origin v8.5.0+1` after merge.
  - GoReleaser workflow runs on tag push; verify Docker image lands as `chatwoot-br/go-whatsapp-web-multidevice:v8.5.0-1` (post-rewrite).
  - `chart-releaser.yaml` publishes Helm chart on `v*` tag.
  - chatwoot-app deployment pin updated to `v8.5.0-1` image; final deploy.
- **Checkpoint:** `docker pull ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:v8.5.0-1` succeeds; `helm pull` of new chart succeeds; chatwoot-app prod is on the new image.
- **Rollback:** Tags are immutable. To "rollback" the release: cut `v8.5.0+2` reverting whatever slice broke, or re-deploy `v8.1.2-1` from registry. Don't `git tag -d` a pushed tag.

## Slice ordering rationale

Slice 1 (reset + smoke) is non-negotiable first — without it, every later
slice's breakage is ambiguous between "reset broke it" and "reapply broke
it." Slice 2 (BR phone) goes next because it touches 39 caller sites across
6 usecase files; pushing it later means doubling sweep effort when Slice 6
revisits the same files. Slice 3 (LID + history_sync_complete) is its own
slice because chatstorage interface changes ripple into the wrapper and any
slice that consumes the storage repo — landing it before Slice 4 keeps the
chatwoot integration's `IChatStorageRepository` consumer (`SyncService` per S)
compiling against the fork-extended interface. Slice 4 (chatwoot lockstep)
sits in the middle: it depends on Slice 3's storage interface and on Slice 2's
phone normalization (chatwoot syncs BR numbers); it gates Slice 5's webhook
audit (the new `is_from_me` behavior affects the consumer); but pushing it
later means longer chatwoot-app PR aging and more cross-repo drift risk.
Slice 5 (taxonomy + env audit) is mostly docs/config — it follows Slice 4
because the `is_from_me` behavior change is a Slice 4 consequence, not a
standalone deltum. Slice 6 (fork sweep + OQ8 files) is last among code
slices because its items are low-risk fork-only re-applies that benefit from
the BR/LID/chatwoot path already being stable. Slice 7 (release) closes.

Vertical-slicing invariant: each slice ships an end-to-end paired-phone
scenario green. Slices 1, 2, 3 are independently shippable as
`v8.5.0+1-rc1`, `-rc2`, `-rc3` if mid-chain risk emerges. Slice 4 is the
ship-or-rollback gate — see risk concentration below.

## Open at P-time

1. **OQ3 file boundary → Option A: `forward_history_sync.go` (new file).**
   Rationale: the reapply diff is one new file rather than a hunk inside
   upstream's restructured 484-line dispatcher; easier to audit and easier to
   submit upstream as a future PR. S confirmed the signature
   (`forwardPayloadToConfiguredWebhooks(ctx, payload, eventName) error`) exists
   on both sides; the choice was purely code-organization.
2. **OQ8+9 fork-heavier files → split.** Usecase files (`chat.go`, `group.go`,
   `message.go`) ride in Slice 2 because the 39-caller BR-phone sweep already
   touches every line that matters there — bundling avoids two passes.
   `device_manager.go` gets its own commit inside Slice 6 (3 fork commits vs
   2 upstream — needs three-way audit). `release.yml` also own-commit inside
   Slice 6. Workflows `chart-releaser.yaml` and `set-latest-tag.yaml` are
   fork-unique (PRESERVED per S) so they don't need an OQ8 decision.
3. **`is_from_me` + `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` deprecation → Slice 4
   (chatwoot-app cutover checklist), not Slice 5.** The behavior change ("all
   outgoing messages now forwarded") affects the consumer the moment chatwoot-app
   cuts over; Slice 5's job is to *document* the dead knob and update the
   taxonomy table, not to handle consumer logic.
4. **Slice 1 paired-phone fixture (NEW OQ).** Slice 1 smoke needs a phone to
   pair against. Three options: (a) throwaway burner phone for the test
   environment, (b) re-pair the existing staging phone (cheap but disruptive),
   (c) checkpoint-restore a session DB from `pre-upgrade-snapshot-2026-05-14`
   into the post-reset build (unclear whether the upstream session-DB schema
   accepts a fork-built session blob — likely yes, since chat-storage schema
   migrations are forward-compatible). **Default: (b) re-pair staging phone**
   — it's a 60-second action and avoids fixture rot.
5. **`phone_br.go` `normalizePhoneBR` visibility (S Open).** Default: keep
   unexported. The BR fixtures live in the same package
   (`phone_br_test.go`), so internal-test access is fine; exporting just to
   satisfy a future external consumer is YAGNI.

## Estimated effort

Rough person-day estimates assuming one engineer with paired-phone access
and staging deploy rights:

| Slice | Days | Risk notes |
|---|---:|---|
| Slice 0 — Preflight | 0.25 | Trivial. |
| Slice 1 — Reset + baseline smoke | 0.5 | One hard reset + paired-phone smoke. If `go test ./...` fails on upstream-vanilla, surface upstream bug back to D — don't paper over with fork reapply. |
| Slice 2 — BR phone + 39 callers + tests | 1.5 | 39 caller sites is the size driver; per-site swap is mechanical but each needs an eyeball pass for upstream signature drift on the call argument. |
| Slice 3 — LID dedup + history_sync_complete | 1.0 | Interface re-add ripples through wrapper + repository; OQ3 file-boundary call already committed (new file). |
| Slice 4 — Chatwoot lockstep cutover | 2.0 | **Highest risk.** Cross-repo PR coordination + staging dry-run + prod cutover window. Day estimate assumes chatwoot-app PR is pre-prepared and reviewed in parallel with gateway PR. If chatwoot-app PR slips, the lockstep window slips. |
| Slice 5 — Webhook taxonomy + env audit | 0.5 | Mostly docs + paired-phone trigger verification. |
| Slice 6 — Fork sweep + OQ8 files | 1.5 | Five sweep items + two own-commit files. `device_manager.go` three-way is the size driver; estimate goes up if upstream's 2 commits touched the fork-edited functions. |
| Slice 7 — Release + CI formalization | 0.5 | CHANGELOG drafting + tag push + workflow regex addition. |
| **Total** | **~7.75** | Single-engineer serial; parallelization possible on Slices 2 and 6, but Slice 4 must serialize against Slice 3. |

**Risk concentrations:**

- **Slice 4 is the ship-stoppable gate.** Lockstep cutover means chatwoot-app
  must be PR-ready, reviewed, and merge-able in the same window as the
  gateway PR. If Slice 4 breaks in prod, both repos roll back together —
  rollback rehearsal in staging is mandatory before prod cutover.
- **Reset+reapply rollback granularity.** Every post-reset slice's rollback
  is "branch reset to `pre-upgrade-snapshot-2026-05-14` and redeploy prior
  image." A mid-chain `git revert` of Slice N forces replay of Slices N+1..M.
  Mitigation: keep slice commits small and atomic so a forced replay is
  cheap; never bundle two slices into one commit.
- **OQ8 `device_manager.go` audit deferred.** If upstream's 2 commits hit the
  same functions fork edited, Slice 6's own-commit balloons to 2-3 commits
  with conflict-resolution. Surface day-of as a Slice 6 sub-risk.
