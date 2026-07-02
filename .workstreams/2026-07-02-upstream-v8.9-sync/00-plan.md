# Upstream sync run: v8.7.0+2 → upstream v8.9.0 (2026-07-02)

Per `docs/upstream-sync.md`. TARGET=8.9.0, N=1 → release `v8.9.0+1`.

## Pre-flight (§3)

- Remotes: origin=chatwoot-br, upstream=aldinokemal ✓
- Merge-base with upstream/main: `131b99b` (2026-06-09, "fix(media): return public URL for downloaded media (#716)")
- Divergence: fork-ahead **65** / upstream-ahead **13**
- Target: `upstream/v8.9.0` (released 2026-06-27). Latest stable upstream tag.
- Partition: 11 commits merge-base→v8.9.0 (**Phase A, this run**); tail v8.9.0→upstream/main is only 2 commits
  (`d6b48dd` whatsmeow bump, `255f727` #748 non-multipart /send/file panic fix) — **skipped**, rides next sync. No Phase B.

## Upstream delta in scope (merge-base → upstream/v8.9.0)

| Commit | What |
|---|---|
| 2221e27, 2daec66, d480900, 70ba814 | whatsmeow bumps: `v0.0.0-20260609091626-4e622162b959` → `v0.0.0-20260622185415-5f04eac6dbbb` |
| a1d617d | fix(chatwoot): pgimport uuid cast — **native chatwoot module, stays dormant** |
| 496e6d7 | #708 stop retrying WA error 463 — **deletes** `send_retry.go`/`send_retry_test.go`, adds `reachout_error.go` (fork never touched send_retry → take upstream) |
| e567365 | #731 persist media `direct_path` — appended migration `ALTER TABLE messages ADD COLUMN direct_path TEXT DEFAULT ''` + column threading in `sqlite_repository.go` |
| 7fd2332 | #740 chat: empty result instead of 500 when chat row absent |
| cf6f2b0 | #735 call-reject API (`POST /call/reject`) — additive endpoint + docs |
| 1b64cb0 | #732 fix random API send timeouts under heavy incoming load |
| d205cff | version bump v8.9.0 |

## Conflict surface (both sides changed since merge-base) — 14 files

- `src/pkg/utils/whatsapp.go` + `_test.go` — **the careful one**: fork BR ninth-digit probe stack vs upstream +101 lines
- `src/infrastructure/chatstorage/sqlite_repository.go` + `_test.go` — mechanical column-list merges; migrations append-only
- `src/infrastructure/whatsapp/history_sync.go` — light (upstream only 5 lines this run)
- `src/usecase/chat.go`, `src/usecase/message.go`, `src/usecase/send.go`
- `src/cmd/root.go`, `src/cmd/rest.go`, `src/config/settings.go`, `src/.env.example`
- `src/go.mod`, `docs/webhook-payload.md`

## Contract-drift expectation (§7)

Upstream webhook changes look **additive** (call.offer `call_id` + reject-call API docs); standard message
webhook shape untouched. Drift note to be written post-merge as `02-contract-drift.md`.

## Hazards touched this run

- Accidental `git fetch upstream --tags` earlier in session created plain local `v8.8.0`/`v8.9.0` tags → deleted
  (twice — branch fetch auto-followed tags once more). Legacy fork `v7.8.x` tags intact. Namespaced
  `upstream/v8.8.0`/`upstream/v8.9.0` fetched per §4.
- whatsmeow API drift is the real risk: gate 1 = `go build ./...` before anything else.

## Gates

1. `go mod tidy` + `go build ./...` → `logs/`
2. `go vet ./...` → `logs/`
3. `go test ./...` → `logs/`
4. contract-drift note
5. release rail: AppVersion `v8.9.0+1`, CHANGELOG, local tag (push = human-gated)
