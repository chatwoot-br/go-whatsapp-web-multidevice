# Pull Request — Sync fork with upstream aldinokemal/v8.5.0

**Target branch**: `chatwoot-br/go-whatsapp-web-multidevice:main` (PR deferred — see Deployment status)
**Source**: local `main` (= former `upgrade/v8.5.0-sync`, now the local trunk per 2026-05-15 decision); old lineage preserved as local `release/v8.1.2` + tag `pre-upgrade-snapshot-2026-05-14`
**Release tag (local)**: `v8.5.0+1`

## Deployment status (2026-05-15)

Local `main` is now the upgrade trunk. **Nothing pushed to origin** — `origin/main` untouched, no release CI fired, no clones broken. Held local by decision until chatwoot integration was confirmed ready; per the contract-drift check (below) it now is. Pushing is a human-owned release decision. Rollback anchors (`pre-upgrade-snapshot-2026-05-14` tag, the old-lineage `release/v8.1.2` branch) are **local-only** — back them to origin under distinct names before any `main` push for off-machine durability.

## Summary

Brings the chatwoot-br fork from `v8.1.2+1` (2026-01-26) to `v8.5.0+1`, syncing 89 upstream commits while preserving all fork-specific deltas. Strategy: Q1=Reset+re-apply (hard-reset to upstream, replay fork commits on top).

## What landed

- **Upstream v8.5.0 baseline**: 89 commits since v8.1.2 (whatsmeow protocol updates ×31, native Chatwoot integration, healthcheck endpoint, LID handling, webhook taxonomy expansion, document thumbnails, ghost mentions, archived chats filter, CTWA Meta Ads, etc.)
- **Fork deltas preserved**: BR phone normalization (39 caller sweep), LID dedup post-history-sync (`deduplicateLIDChats` + `MergeLIDChat` + `GetLIDChats` + `NormalizeJIDFromLIDWithContext`), `history_sync_complete` event, proxy support (SOCKS5/HTTP/HTTPS), info-request cache, S3 Content-Type extension fix, `chat_name`/`sender_name` webhook payload fields, OQ8 `device_manager.go` proxy hooks
- **8 CHATWOOT_* env vars** wired through Helm + configmap + secret — **dormant** (CHATWOOT_ENABLED defaults false). The upstream native chatwoot module arrived via the v8.5.0 reset but is NOT activated: Q2 was reversed (`10-q2-reversal.md`) — the fork keeps its own custom bidirectional chatwoot integration. Wiring kept (not reverted) to avoid re-introducing rebase divergence; it is inert.
- **Webhook contract**: union event taxonomy documented in `docs/webhook-payload.md`; `is_from_me` payload field documented; `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` marked deprecated
- **Test coverage**: ~25 new test files / scenarios covering BR phone, LID dedup, history_sync_complete dispatch, HMAC signing, chat_name lookup, Chatwoot integration with mocked HTTP, e2e cross-package integration. Tests caught and fixed a Slice 3 regression in `GetLIDChats`.
- **CI**: release.yml has a `vX.Y.Z+N` tag-shape regex gate so future syncs can't drift on naming.

## Slice-by-slice diff

- **Slice 1** — `95166f7` reset to upstream/v8.5.0 + smoke + release rail reapply
- **Slice 2** — `284852b` BR phone normalization layer (`phone_br.go`) + 39 caller sweep; merge `1e32c83`
- **Slice 3** — `4b6ac0d` LID dedup primitives + `history_sync_complete` event dispatch; merge `6c14b5f`; reconcile `589292e` routes `@lid` through `ResolveLIDToPhone`
- **Slice 6** — fork-only delta sweep (`9796adf` S3 Content-Type extension fix, `bd660c7` info_cache infrastructure, `24891a8` proxy SOCKS5/HTTP/HTTPS, `e7a6848` full history sync + ON_DEMAND); merge `d314e39` + OQ2 reconcile `ef18ec8` drops dead `NormalizeJIDFromLID`
- **Slice 4** — `c11801e` chatwoot lockstep cutover (gateway-side wiring, 8 env vars + Helm)
- **Slice 5** — `1426149` webhook taxonomy + env audit; `25f494b` recover `chat_name` + `sender_name` payload fields
- **Slice 7** — `f6f0f52` release v8.5.0+1 (CHANGELOG + AppVersion bump + CI tag-gate regex)
- **Phase 7 (e2e)** — merge `238ee97`: 9 new test files (BR phone, LID dedup primitives, history_sync_complete + LID-nil branches, HMAC, chat_name, chatwoot mocked HTTP, info_cache TTL, cross-package e2e under `//go:build e2e`); caught + fixed Slice 3 regression `aa1c21e` (`GetLIDChats` SELECT missing `archived` column)

## Known follow-ups (NOT in this PR)

1. **chatwoot-app Rails-side cutover — NO LONGER REQUIRED.** Q2 was reversed after the topology investigation (`09-chatwoot-app-topology.md`) showed GoWA-native chatwoot is a 5-feature regression vs the fork's integration and the fork's WhatsApp channel has zero rebase-surface tax. The fork keeps its custom integration. The webhook contract-drift check (`11-contract-drift.md`) confirms the post-v8.5.0 GoWA *standard* webhook is **contract-clean** against chatwoot-app's fork controller: 0 breaking, 1 behavioral (`is_from_me` outgoing-echo — already absorbed by chatwoot-app's existing `source_id` dedup, independent of the gate `3b87f4e` deleted), HMAC stable. **No chatwoot-app code change needed.** Optional deploy-config only: whitelist `WHATSAPP_WEBHOOK_EVENTS` to suppress `chat_presence` log/Sidekiq churn (~2.5h, no code).
2. **Paired-phone validation**: trigger each event from staging phone; confirm receipt at test webhook with documented payload shape. See `docs/webhook-payload.md` § Verification scenarios (paired-phone). One specific item to confirm empirically: message-id identity across send→echo (the `source_id` dedup that makes outgoing-echo safe assumes it; whatsmeow norm, not traced — UAT item 1).
3. **Cache consumer wiring**: Slice 6 landed `info_cache` infrastructure but not consumer wiring in `src/usecase/user.go` + `group.go` (deferred to avoid Slice 2 territory overlap). Trivial follow-up.
4. **Upstream PR opportunity**: fork's BR phone rules could be submitted upstream as a small contribution to `src/pkg/utils/phone.go`. Would shrink fork divergence.

## Test plan

- [x] `go test ./...` green
- [x] `go test -tags=e2e ./e2e/...` green
- [x] `helm lint charts/gowa` clean
- [ ] Paired-phone validation (human gate, separate PR-review step)
- [x] chatwoot-app webhook contract verified clean (`11-contract-drift.md`) — no Rails PR needed (Q2 reversed)
- [ ] (optional) chatwoot-app deploy: whitelist `WHATSAPP_WEBHOOK_EVENTS` to drop `chat_presence` noise

## Coverage snapshot (post-merge)

| Package | Coverage |
|---|---|
| `pkg/utils` | 34.6% |
| `infrastructure/chatwoot` | 36.4% |
| `infrastructure/chatstorage` | 13.6% |
| `infrastructure/whatsapp` | 12.1% |

Lower whatsapp/chatstorage % is expected — those packages contain large amounts of WhatsApp client glue and SQL that requires a live device or DB for meaningful coverage; the tests focus on fork-specific primitives (LID dedup, history_sync_complete, info_cache, BR phone routing, chat_name lookup) which are individually well-covered.

## Workstream artifacts

- `.workstreams/2026-05-14-upstream-v8.5-sync/` carries the full QRSPI trail: Q/R/D/S/P/W stage docs, two parallel P-stage drafts (Claude+Codex) + consolidation rationale, execution plan with 8-phase tracking, logs from every phase. Useful for code review and for the next sync.

## Rollback

Tag `pre-upgrade-snapshot-2026-05-14` preserves the pre-upgrade fork state. To roll back: redeploy the prior Docker image AND `git reset --hard pre-upgrade-snapshot-2026-05-14`.
