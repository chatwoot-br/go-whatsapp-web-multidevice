# Upstream sync run: v8.9.0+1 → upstream v8.10.0 (2026-07-10)

Per `docs/upstream-sync.md`. TARGET=8.10.0, N=1 → release `v8.10.0+1`.

## Pre-flight (§3)

- Remotes: origin=chatwoot-br, upstream=aldinokemal ✓
- Merge-base with upstream/main: `d205cff` (2026-06-27, "chore: bump version to v8.9.0") — exactly the v8.9.0 sync point
- Divergence: fork-ahead **84** / upstream-ahead **21**
- Target: `upstream/v8.10.0` (= `e4c62f8`, released 2026-07-03). Latest stable upstream tag.
- Partition: 14 commits merge-base→v8.10.0 (**Phase A, this run**); tail v8.10.0→upstream/main is 7 commits
  (per-device login endpoints, WHATSAPP_PROXY #664, WHATSAPP_WEBHOOK_IGNORE_JIDS #736, send-link fix #661,
  chat composer UI, forward-by-ID #755, whatsmeow 20260709 bump) — **skipped**, rides next sync. No Phase B.
- **Unlike v8.9.0, chatwoot/webhook changes are INSIDE the tag this run** (per-device webhooks #671 + fixes,
  chatwoot @newsletter guard) → contract-drift check (§7) is mandatory in Phase A.

## Upstream delta in scope (merge-base → upstream/v8.10.0, 14 commits, 58 files +4552/−525)

| Commit | What |
|---|---|
| 255f727 | #748 don't panic on non-multipart POST /send/file |
| d6b48dd | whatsmeow bump `20260622185415-5f04eac6dbbb` → `20260630180629-b572e5bcb92b` (+gqlparser 2.5.36) |
| 5072224 | #749 newsletter latest-messages endpoint |
| 5aea840, b58c932 | #722 MCP send/manipulation tools; persist incoming location messages; tests |
| 4d39e10 | #728 logout disconnects session instead of deleting device entry (Hub Operacional — our upstreamed fix) |
| ca43bb5 | MCP handlers → mcp-go request helpers |
| a66f883 | #754 passkey pairing support |
| 0de0466, 23620dc, e4c62f8 | #671 **per-device webhook support** + review-gap fixes + broadcast-skip/no-rows fixes |
| e3aac4c | chatwoot: skip @newsletter JIDs before contact creation (native module — stays dormant) |
| 8b73125, fbfc99a | docs; version bump v8.10.0 |

## Conflict surface (both sides changed since merge-base) — 20 files

- `src/infrastructure/whatsapp/event_handler.go` — fork: detached dispatcher context (1300040/495c632); upstream: per-device webhook + passkey events (+106)
- `src/infrastructure/whatsapp/event_message_handler.go` — fork: `chat_name`/`sender_name` fields; upstream +25
- `src/infrastructure/whatsapp/device_manager.go`, `device_instance.go`, `init.go` — fork: proxy + `RequireFullSync`; upstream: per-device webhook plumbing + #728 logout
- `src/infrastructure/whatsapp/chatstorage_wrapper.go`, `src/domains/chatstorage/interfaces.go`,
  `src/infrastructure/chatstorage/sqlite_repository.go` + `_test.go` — fork: LID-dedup methods; upstream: per-device webhook storage (migrations **append-only**)
- `src/infrastructure/chatwoot/sync.go` — fork: direct_path-only media gate (68df5af); upstream: @newsletter JID skip — reconcile both, module stays dormant
- `src/pkg/utils/whatsapp.go` + `_test.go` — fork BR-phone stack; the careful one every run
- `src/usecase/device.go` — upstream #728 + per-device webhook CRUD
- `src/usecase/newsletter.go` — upstream #749
- `src/config/settings.go`, `src/views/components/DeviceManager.js`, `src/infrastructure/whatsapp/event_label_test.go`,
  `src/go.mod`, `docs/webhook-payload.md`, `readme.md`

## Contract-drift expectation (§7) — MANDATORY this run

Per-device webhooks (#671) change webhook *routing* (per-device URLs/config) and possibly payload/delivery
(broadcast skip in e4c62f8). Must verify `forwardToWebhooks` output still matches chatwoot-app's
`whatsapp_web_controller.rb` contract: top-level `device_id`/`event`/`payload`, event switch incl.
fork-custom `history_sync_complete`, `chat_name`/`sender_name`, HMAC `X-Hub-Signature-256`.
Drift note → `02-contract-drift.md`.

## Hazards touched this run

- Accidental `git fetch upstream --tags` early in session created plain local `v8.10.0` tag → deleted
  (same footgun as v8.9 run). Legacy fork `v7.8.x` tags intact. Namespaced `upstream/v8.10.0` fetched per §4.
- whatsmeow API drift: gate 1 = `go build ./...` first.
- §7.5 audit focus: per-device webhook code is new upstream surface touching device scoping — sweep
  `usecase/` for non-`*ByDevice` chat/message reads; `ChatStorageMaxOpenConns` must stay 1; check no
  fork bound/retry dropped.

## Gates

1. `go mod tidy` + `go build ./...` → `logs/`
2. `go vet ./...` → `logs/`
3. `go test ./...` → `logs/`
4. Post-merge audit (§7.5): device-scoping sweep, default-flip diff, whatsmeow-workaround check, fork-delta review + `/code-review`
5. Contract-drift note (§7) → `02-contract-drift.md`
6. Release rail: AppVersion `v8.10.0+1`, CHANGELOG, local tag (push = human-gated)
