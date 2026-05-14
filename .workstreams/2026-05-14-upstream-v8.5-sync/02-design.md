# D — Design (~200 lines target)

QRSPI D stage. Produced from R + answered Qs. **Reviewed by the code owner before any plan exists.**

Borrowed from Matt Pocock's "design concept" framing — capture decisions before code, not after.

## Current state

Fork `main` (`b4fd010`, tagged `v8.1.2+1`) sits **41 commits ahead** of upstream `v8.1.2` (`097403d`) and **88 commits behind** `upstream/main` (`17af98e`, `v8.5.0`+5). Last common ancestor: `48d9be8`. Total churn in `main..upstream/main`: 134 files, +6841/−8830. The 37-file overlap zone is **upstream-heavier on 23 files** (notably `src/pkg/utils/whatsapp.go` 11 vs 2, `src/usecase/send.go` 8 vs 4, `src/config/settings.go` 14 vs 9, `src/go.mod` 35 vs 4) and **fork-heavier on 8** (`history_sync.go` 5 vs 2, `device_manager.go` 3 vs 2, the GitHub Actions workflows, `.gitignore`). Upstream now ships a chatwoot integration the fork lacks entirely: `src/infrastructure/chatwoot/{client,sync,sync_test,sync_types,types}.go`, `src/ui/rest/chatwoot.go`, and a 484-line expansion of `webhook_forward.go` introducing `forwardToChatwoot`, `forwardToWebhooks`, contact/group caching, and a chatwoot allow-list on `message`/`message.reaction` events.

Fork deltas the reset must re-apply (CHANGELOG, oldest → newest):

- **v8.1.0+1** — Helm chart at `charts/gowa/`; `chat_name`, `sender_name`, `is_from_me` on the outgoing webhook payload; `history_sync_complete` event; OGG Opus PTT conversion; audio waveform + MIME/duration resolution; `APP_BASE_PATH` middleware fix; debounced history-sync webhook; multi-device docs; `+N` versioning + chart-releaser CI.
- **v8.1.0+2** — Short-term cache for info requests (incl. error caching); LID-based send path; webhook event list extended (`history_sync_complete`); various CI tag-pattern + multi-arch fixes; Helm release moved to `v*` tags only.
- **v8.1.0+3** — SOCKS5/HTTP/HTTPS proxy support with proxy IP rendered in the device-card UI; webhook caption included on auto-downloaded media.
- **v8.1.0+4** — Full history sync + ON_DEMAND capability; ON_DEMAND response handling; unavailable-message handling from linked devices; `chat_id` normalized from LID to phone in webhook; logs dir ignored.
- **v8.1.0+5** — S3 image-extension fix derived from Content-Type.
- **v8.1.0+6** — `MergeLIDChat` + `NormalizeJIDFromLIDWithContext` + post-history-sync LID dedup (`deduplicateLIDChats` in `history_sync.go`).
- **v8.1.0+7** — `ValidateAndNormalizeJID` (BR 9-digit mobile normalization) and its rollout across send/chat/group/message/newsletter/user usecases; `ValidateJidWithLogin` deprecated. Note: function lives in `src/pkg/utils/whatsapp.go:670`, not `general.go` (R prompt was off-by-one file).
- **v8.1.2+1** — Sync with upstream `v8.1.2`; whatsmeow bump to `v0.0.0-20260126173513-4dbbef8d4d4a`; `mailcap` added to the Docker image for MIME types.

Pre-upgrade snapshot lives at tag `pre-upgrade-snapshot-2026-05-14` (per Q1 rationale). Fork-unique CI: `chart-releaser.yaml`, `set-latest-tag.yaml`, plus the `+`→`-` tag rewrite in `build-docker-image.yaml:24` to make `v8.1.2+1` a legal Docker tag.

## End state

After Q1+Q2+Q3+Q4 ship, `main` HEAD is `upstream/main` (`17af98e`) **plus ~9–11 fork-delta commits** on top — one per surviving `+N` feature area, not one per historical `+N` (some collapse, some split):

- `feat(helm): vendor charts/gowa Helm chart`
- `ci: +N versioning, Docker tag rewrite, chart-releaser, set-latest workflows`
- `feat(audio): OGG Opus PTT conversion + waveform + MIME/duration`
- `feat(proxy): SOCKS5/HTTP/HTTPS proxy + UI proxy-IP display`
- `feat(cache): short-term info-request cache with error caching`
- `feat(webhook): history_sync_complete event + debounce + LID→phone chat_id normalization` (decide at P-time whether this folds into upstream's expanded event set)
- `fix(utils): derive S3 image extension from Content-Type`
- `fix(history-sync): MergeLIDChat dedup + NormalizeJIDFromLIDWithContext` (may shrink — see Open questions)
- `feat(phone-br): phone_br.go layering BR 9-digit rules over upstream pkg/utils/phone.go` (Q3)
- `fix(docker): mailcap for MIME types`
- `chore(deps): whatsmeow bumps beyond upstream HEAD, if any` (likely empty at cut)

Native chatwoot integration lands intact under `src/infrastructure/chatwoot/{client,sync,types}.go` and `src/ui/rest/chatwoot.go`. The upstream gateway runs in **push** mode: on each `message` / `message.reaction` event, `forwardToChatwoot` (introduced in `webhook_forward.go` per `44a128c`, refined by `3b87f4e` and `909b6e6`) calls Chatwoot's REST API directly with allow-listed event types, contact resolution cached behind `contactMutexShards` + `groupNameCache`, and a synthesized message-content builder (`buildChatwootMessageContent`, `buildReactionChatwootContent`, `extractStructuredMessageContent`). `chatwoot-app` (the Rails consumer) is **cut over from listener to receiver**: `Channel::Whatsapp::Provider`'s `process_messages` path stops parsing the legacy custom JSON shape from `/webhook` and instead exposes its existing `/webhooks/whatsapp/:hmac_token` endpoint to consume the upstream gateway's chatwoot-aware payload; the bespoke webhook-shape parser (the `format_status_message` / `process_messages` branch matching the fork's payload keys `chat_name`, `sender_name`, `history_sync_complete`) is retired in the same change. New env knobs (`CHATWOOT_ENABLED`, `CHATWOOT_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`, `CHATWOOT_INBOX_ID`, `CHATWOOT_DEVICE_ID`, `CHATWOOT_IMPORT_MESSAGES`, `CHATWOOT_DAYS_LIMIT_IMPORT_MESSAGES`, `CHATWOOT_IMPORT_CONTACTS`) flow through into the gateway's deployment manifests and Helm values; fork's only env-line edit (`history_sync_complete` appended to `WHATSAPP_WEBHOOK_EVENTS`) is reapplied on the new default list.

The `## Available Webhook Events` table in `docs/webhook-payload.md` reflects the union of upstream's additions (`chat_presence` from `c428afa`; `call.offer` from `5c193bc`) and the fork's existing `history_sync_complete`. The `is_from_me` field upstream added under `### Common Payload Fields` (`44a128c` and follow-ups) replaces the fork's hand-rolled `is_from_me` from v8.1.0+1 — fork's caller side already populates the field, so this is a no-op behaviorally; the schema page stops being a fork divergence.

`src/pkg/utils/phone.go` is upstream's (generic). A new `src/pkg/utils/phone_br.go` wraps the upstream entry points with BR 9-digit normalization (the logic currently inlined in `ValidateAndNormalizeJID` at `src/pkg/utils/whatsapp.go:670–730`, which queries `client.IsOnWhatsApp` and returns whatsmeow's canonical JID). Existing `ValidateAndNormalizeJID` callers in `usecase/{send,chat,group,message,newsletter,user}.go` (`559936e`) route through the layered function — exact entry-point shape determined at P-time, but the contract on the call sites stays `(client, jid) → (types.JID, error)` to keep the reapply diff to one file + the new `phone_br.go`. The deprecated `ValidateJidWithLogin` is removed in the same commit. The Helm chart in `charts/gowa/{Chart.yaml,values.yaml,templates/,README.md}` still ships from the fork via the existing `chart-releaser.yaml` workflow — no upstream Helm equivalent has appeared.

Version tag at cut: `v8.5.0+1` (see Open questions on `+0` vs `+1`). Pre-upgrade tag `pre-upgrade-snapshot-2026-05-14` retained indefinitely for archaeology.

## Patterns to follow

Conventions that must survive the reset+reapply:

- **Versioning suffix `vX.Y.Z+N`** — fork increments `+N` while base `vX.Y.Z` tracks the upstream tag synced against. CHANGELOG `## [v8.1.2+1] - 2026-01-26 (Synced with upstream v8.1.2)` is the canonical header shape.
- **CHANGELOG section layout** — `### Upstream Changes` first (one bullet per upstream commit pulled in), then `### Fork Changes`. For the cut, both sections will be unusually large; keep the split.
- **Docker tag rewrite** — `.github/workflows/build-docker-image.yaml:24` rewrites `+` → `-` for OCI compatibility (`v8.1.2+1` → `v8.1.2-1`). The release workflow at `.github/workflows/release.yml` triggers on `v*` tags and runs GoReleaser. Both must survive untouched.
- **Helm release path** — `.github/workflows/chart-releaser.yaml` triggers on `v*` and publishes from `charts/`; chart sources stay at `charts/gowa/{Chart.yaml,values.yaml,templates/,README.md}`.
- **Latest-tag promotion** — `.github/workflows/set-latest-tag.yaml` is a manual `workflow_dispatch` gate; deliberate, not automatic.
- **Branch naming** — workstream branches sit under `.workstreams/YYYY-MM-DD-<slug>/`; the cut itself lands on `main` after review.
- **`WHATSAPP_WEBHOOK_EVENTS` default** — the fork's default event list includes `history_sync_complete`; the post-reset default must keep it (unless Open question on event-list reconciliation resolves the other way).

## Resolved decisions

- **Q1 (upgrade strategy): Reset + re-apply.** Cleanest divergence story; each fork delta becomes a self-contained commit PR-able upstream individually; pre-reset history archived at `pre-upgrade-snapshot-2026-05-14`.
- **Q2 (chatwoot integration): Adopt + migrate.** Upstream now owns the chatwoot contract; staying webhook-only means re-implementing what upstream maintains in perpetuity; chatwoot-app cuts over to upstream's API after Slice 4 lands.
- **Q3 (phone normalization): Layer.** Upstream's `pkg/utils/phone.go` is the base; fork ships `phone_br.go` as a thin BR-specific override; single-file divergence sets up a future upstream PR (Option A) cheaply.
- **Q4 (scope): All-in.** Reset+re-apply makes "all-in" free relative to selective cherry-pick; consumers of new features (CTWA, GIF, ghost mentions, archived-chats filter, document thumbnails) light up automatically.

## Open questions

- **`3b87f4e` webhook-auth-fix scope** (R-flagged `(needs decision)`). Commit subject says "webhook auth fix" but R confirmed `webhook.go` HMAC path is untouched on both sides; commit body scopes the fix to `src/ui/rest/chatwoot.go` and `src/cmd/rest.go` (the chatwoot REST surface, not the outgoing webhook signing). D-stage assumption: this fix is a chatwoot-REST authn change, not a fork-facing HMAC break. Confirm before P-stage writes the Slice 4 chatwoot-app cutover — if the fix touches signature verification on the inbound chatwoot side, the chatwoot-app HTTP client config has to change too.
- **`MergeLIDChat` survival under Q3.** Q3 covers phone-normalization only; `MergeLIDChat` + `deduplicateLIDChats` + `NormalizeJIDFromLIDWithContext` (v8.1.0+6) sit in `src/infrastructure/whatsapp/history_sync.go` and `chatstorage` repo interfaces — outside Q3's scope. Upstream's 3 LID-handling commits (`40b0875`, `d718ef8`, `17ff32f`) may or may not subsume the fork's post-sync dedup pass. Need a line-level read of those three upstream commits before the reapply step. If they cover it: drop the fork delta. If not: keep `deduplicateLIDChats` as its own re-applied commit.
- **`history_sync_complete` event under upstream's expanded event taxonomy.** Upstream added `chat_presence` and `call.offer`; fork added `history_sync_complete`. None overlap, so no collision, but upstream's `webhook_forward.go` rewrite (+484 lines) restructures `forwardToWebhooks`. The fork's debounced history-sync notification has to plug into the new dispatcher. Decide at P-time whether this is a new file or an edit on upstream's `webhook_forward.go` — affects how clean the reapply commit looks.
- **`v8.5.0+0` vs `v8.5.0+1` tag at cut.** Carried over from Q-stage Open questions; resolves in P-stage when the first tag is cut. Default assumption: `+1`, since fork deltas (Helm, proxy, audio, BR phone, etc.) ride on top of upstream.
- **Coordinated chatwoot-app cutover vs independent gateway ship.** Carried from Q-stage. If chatwoot-app's `Channel::Whatsapp::Provider` can dual-listen (accept both the legacy webhook shape and the new `/chatwoot/forward` push) during a window, the gateway can ship first; otherwise Slice 4 has to land lockstep. P-stage gating decision.
- **In-flight customer integrations on the current webhook payload shape.** Carried from Q-stage. New top-level field `is_from_me` under `### Common Payload Fields` is additive (safe); `chat_presence` and `call.offer` are new event names (gated by `WHATSAPP_WEBHOOK_EVENTS`, so off-by-default for existing tenants). Inventory of customer consumers needs to happen out-of-band — D-stage cannot answer.
- **Fork-unique CHANGELOG entries `381c381` ("Persist incoming contact messages in chatstorage") and `a8b5ed8` ("Persist incoming calls to chat storage").** R noted these have no conventional prefix and land in none of the 7 buckets. They're upstream commits in `main..upstream/main`, not fork commits — so they ride in automatically under Q4=all-in. Flag for the reviewer in case "uncategorised" means "unexamined."
- **`src/usecase/group.go` collision (fork-heavier, 3 vs 2).** Q3's BR-phone callers run through every entry of `usecase/group.go` (10 call sites of `ValidateAndNormalizeJID` per current `grep`). Upstream has 2 commits touching this file — likely feat work, not callsite changes. If upstream's edits aren't in the same functions, the reapply is clean. If they are, P-stage gets a conflict-resolution slice. Settle before the reapply commit lands.
- **`src/infrastructure/whatsapp/device_manager.go` (fork-heavier 3 vs 2) and `src/usecase/{chat,group}.go` (fork-heavier 3 vs 2 each).** These are the four overlap files where the fork has divergent edits upstream hasn't touched comparably. Treat each as a per-file reapply where R-stage hasn't already shown the upstream diff is benign.
- **Test reapply.** R surfaced 3 fork-side test files (`jid_utils_test.go`, `general_test.go`, `whatsapp_test.go`); upstream touches `general_test.go` and `whatsapp_test.go` too. After Q3 layering, the BR-specific tests in `jid_utils_test.go` need to move (or stay) against `phone_br.go` rather than the deprecated `ValidateJidWithLogin` callsite. Confirm test surface at P-time.

## Non-goals

Explicitly out of scope for this upgrade:

- **`chatwoot-app` schema migrations.** The Rails side may need DB shape changes to consume upstream's chatwoot API. Separate workstream, separate review. This upgrade ships the gateway-side adoption and the Rails-side HTTP-call switch only.
- **Upstreaming `phone_br.go` (Option A in Q3).** Future PR against `aldinokemal/go-whatsapp-web-multidevice` once the layered override stabilises. Not in this cut.
- **Monthly auto-sync CI gate.** A scheduled job that opens upgrade PRs when `upstream/main` advances has been discussed; not in this workstream.
- **Unrelated dependency upgrades** beyond what `upstream/main` already pulls in via the 35 commits touching `src/go.mod` / 34 touching `src/go.sum`. No fork-side bumps in this cut.
- **Retiring the fork's `history_sync_complete` event** even if upstream's expanded taxonomy could carry equivalent signal. Backward-compat for existing webhook consumers; revisit in a later sprint with a deprecation window.
- **Refactoring fork-heavier overlap files** (`history_sync.go`, `device_manager.go`, the workflows) into "cleaner" shapes during reapply. Reapply preserves behaviour; refactors are separate commits or separate sprints.
- **Touching `pre-upgrade-snapshot-2026-05-14`.** It is an immutable archaeology tag. Do not rebase, force-push, or delete.
- **Renaming, splitting, or relocating the Helm chart.** It stays at `charts/gowa/` with the existing `chart-releaser.yaml` flow.
