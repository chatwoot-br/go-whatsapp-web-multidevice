# Upstream sync plan — chatwoot-br/go-whatsapp-web-multidevice → upstream v8.7.0

Date: 2026-06-11. Author: orientation pass (read-only investigation + plan draft).
Status: **DRAFT** — not executed. Remotes fetched; no branches created, nothing pushed.

## 0. TL;DR

Fork `origin/main` is **42 commits ahead / 39 behind** upstream `main` (merge-base
`17af98e`, 2026-05-14, just past upstream v8.5.0). Upstream has shipped
v8.5.1 → v8.6.0 → **v8.7.0** plus 12 unreleased commits on `main`.

Recommended approach: **merge (not rebase)** upstream **tag `v8.7.0`** into a new
`upgrade/v8.7.0-sync` branch off `origin/main`, resolve once, PR to `origin/main`.
The 12 unreleased commits past v8.7.0 — which carry **all** the chatwoot + webhook-shape
changes — are a **separate, later Phase B**.

This mirrors the proven house playbook from `.workstreams/2026-05-14-upstream-v8.5-sync/`.

## 1. Current state (verified)

| Fact | Value |
|---|---|
| Remotes | `origin` = github.com/chatwoot-br/… (fork), `upstream` = github.com/aldinokemal/… |
| Fork tip `origin/main` | `4ece676 chore(release): v8.5.0+5` |
| Local `main` | ahead of `origin/main` by 1 (`adf605c chore: ignore graphify-out`) — fork housekeeping, not part of this sync |
| Merge-base | `17af98e` (2026-05-14, upstream "chore: update whatsmeow to latest") |
| Divergence | origin/main **+42**, upstream/main **+39** vs merge-base |
| Latest upstream stable | **v8.7.0** (`cf8b1b4`, 2026-06-05) |
| Upstream `main` tip | `131b99b` (12 commits past v8.7.0, unreleased) |
| `origin/upstream` branch | **stale mirror**, 137 behind upstream/main (last 2026-01-05). Ignore — track `upstream/main`. |

### Tag-collision wrinkle (standing repo hazard)

The fork **re-used upstream's version-tag names**. Local `v7.8.0`/`v7.8.2` point to the
fork's own release commits (also tagged `gowa-7.8.0`/`gowa-7.8.2`); upstream's same-named
tags point to different commits. A plain `git fetch upstream --tags` **fails** with
"would clobber existing tag". During this orientation, upstream tags were fetched into a
**non-colliding namespace** instead:

```
git fetch upstream "refs/tags/*:refs/tags/upstream/*"   # → refs/tags/upstream/v8.7.0, etc.
```

→ Reference upstream releases as `upstream/v8.7.0` (the namespaced tag) throughout this plan.
Do **not** force-overwrite the fork's `v7.8.x` tags.

## 2. What's coming from upstream

### v8.7.0 and below (Phase A target — chatwoot-free)

Protocol/core/dependency churn. Dependency deltas (`src/go.mod`):

| Dep | Fork (v8.5.0+5) | v8.7.0+ |
|---|---|---|
| **go.mau.fi/whatsmeow** | `…20260513140310` (May 13) | `…20260609091626` (Jun 9) — **~1 month of protocol churn** |
| go (toolchain) | 1.25.0 | 1.25.5 |
| gofiber/fiber/v2 | 2.52.12 | 2.52.13 |
| valyala/fasthttp | 1.69.0 | 1.71.0 |
| go.mau.fi/libsignal | 0.2.1 | 0.2.2 |
| **modernc.org/sqlite** | — | **NEW** v1.50.1 (pure-Go sqlite, build-tagged armv6/7 — upstream `61a7a50`) |
| mark3labs/mcp-go | 0.45.0 | 0.54.0 |

Notable feature/fix commits ≤ v8.7.0: quoted media replies (#705), persist message
edit history (#679), persist reactions, scheduled presence pulse (#692), WhatsApp 463
send-failure mitigation (#695), SecretEncryptedMessage edit decrypt (#693), preserve
store identity on restart (#706), ARMv7 + pure-Go sqlite build path.

### The unreleased tail (Phase B — chatwoot + webhook shape)

`git log upstream/v8.7.0..upstream/main` = 12 commits. **All chatwoot/webhook-contract
work lives here**, which is exactly why it is partitioned out of Phase A:

- `1316f70 feat(api): add session_id to webhooks and jid to app responses` — **touches webhook payload shape → contract-drift risk**
- `77db428 feat: chatwoot integration updates`
- `9266165 fix(chatwoot): use saved WhatsApp contact name instead of phone number`
- `319c67e fix(chatwoot): trim whitespace from API token and URL`
- `23a8de2 fix(chat): fall back chat-list name to phone number when empty`
- plus `131b99b` media-URL fix, 2× whatsmeow bumps, docs, release-note skill.

## 3. Why merge, not rebase

- The **no-amend / no-force-push** house rule (squash at PR merge) rules out rebasing 42 fork commits — that rewrites SHAs and forces a push.
- Merge resolves the divergence **once**; rebase replays 42 commits = up to 42 conflict points.
- Last sync (v8.5) jumped a **major line** (v7.8 → v8.5) and warranted a curated reset+reapply. This is a **same-major** step (v8.5.0+5 → v8.7.0); a straight merge is appropriate.
- **Do not reopen** the GoWA-native-chatwoot question: the fork settled it last sync (Q2 reversal — native module stays **dormant**, `CHATWOOT_ENABLED=false`; the fork's own integration + standard webhook path is canonical). See `.workstreams/2026-05-14-upstream-v8.5-sync/10-q2-reversal.md`.

## 4. Conflict surface (Phase A, vs `upstream/v8.7.0`)

**28 files** changed by **both** the fork and v8.7.0 since the merge-base — **zero are chatwoot
files** (the only chatwoot overlap, `client_test.go`, is a post-v8.7.0 change and defers to
Phase B). Buckets:

- **Highest risk — `src/infrastructure/whatsapp/history_sync.go`**: fork **+375/−62**, upstream **+49/−13** on the same file. Fork rewrote full-history-sync + ON_DEMAND + forwarding; must re-resolve over upstream's edits **and** against the newer whatsmeow API.
- **whatsapp event/core**: `event_message.go`, `event_message_handler.go`, `database.go`, `device_manager.go`, `chatstorage_wrapper.go`, `init.go` (+ their `_test.go`).
- **phone/JID utils**: `pkg/utils/whatsapp.go`, `general.go` (fork's BR phone normalization + LID dedup lives adjacent) + tests.
- **chatstorage**: `sqlite_repository.go`, `domains/chatstorage/interfaces.go`, `domains/chat/chat.go`.
- **usecase**: `chat.go`, `send.go`.
- **config/CLI**: `config/settings.go`, `cmd/root.go`, `cmd/rest.go`, `src/.env.example` (mostly append-only — proxy/cache/webhook-event vars).
- **mechanical**: `readme.md`, `docs/webhook-payload.md`, `.github/workflows/{build-docker-image,release}.yml`, `go.mod`/`go.sum`.

**The real risk is not text conflicts — it is whatsmeow API drift.** The fork's event &
history-sync layer calls deep into whatsmeow; a month of upstream changes there may break
compilation/behavior even where the text merges cleanly. `go build` + `go test` against the
**new** whatsmeow is the load-bearing gate, not conflict resolution.

## 5. Phased plan

### Phase A — sync to v8.7.0 (chatwoot-free, primary deliverable)

| # | Step | Exit gate |
|---|---|---|
| A0 | From `origin/main`, branch `upgrade/v8.7.0-sync`. (Fold in or set aside local `adf605c` housekeeping commit.) | branch exists |
| A1 | `git merge upstream/v8.7.0` (the namespaced tag). Expect conflicts in the 28 files §4. | merge in progress |
| A2 | Resolve conflicts. Order: utils/JID → chatstorage interfaces → event layer → **`history_sync.go`** → usecase → config/docs/CI. Preserve fork features: BR phone normalization (`phone_br.go`), LID dedup, `history_sync_complete` webhook event, info cache, SOCKS/HTTP proxy, HMAC `X-Hub-Signature-256` signing, `chat_name`/`sender_name` webhook fields. | all conflicts resolved |
| A3 | `cd src && go mod tidy` — reconcile go.mod/go.sum incl. new `modernc.org/sqlite`, bumped whatsmeow/fiber/fasthttp/libsignal, go 1.25.5. | tidy clean |
| A4 | **`cd src && go build ./... && go vet ./...`** against new whatsmeow — fix API drift in the fork's whatsmeow call sites. | builds clean |
| A5 | **`cd src && go test ./...`** — fix fork tests broken by upstream behavior/signature changes. | all green |
| A6 | Release-rail bump: `v8.7.0+1` (fork's `vX.Y.Z+N` scheme), CHANGELOG, AppVersion. **Local tag only — not pushed.** | tag created locally |
| A7 | Push `upgrade/v8.7.0-sync`, open PR → `origin/main` with this doc + resolution notes. | PR open |

### Phase B — unreleased tail (optional / second step, separately validated)

Do **after** Phase A lands. Merge `upstream/main` (the 12-commit tail) into a fresh
`upgrade/v8.7.x-tail-sync` branch. This is where chatwoot reconciliation happens:

- Reconcile upstream's chatwoot edits (`77db428`, `9266165`, `319c67e`) against the fork's
  custom `infrastructure/chatwoot/` — keep the fork's integration canonical, native dormant.
  Note: upstream's chatwoot package is now **larger** than the fork's (it added `pgimport/`,
  `provision.go`, `ui/rest/chatwoot.go`, `pkg/utils/chatwoot.go`) — decide per-file whether to
  absorb or leave dormant; **do not** let it silently re-route the live webhook path.
- **Re-run the webhook contract-drift check** (the load-bearing chatwoot gate) — `1316f70`
  adds `session_id` to webhook payloads. Re-validate GoWA's standard `forwardToWebhooks`
  output still matches chatwoot-app's `Webhooks::WhatsappWebController`. Template:
  `.workstreams/2026-05-14-upstream-v8.5-sync/11-contract-drift.md`.
- Same build/test/release-rail gates as Phase A.

## 6. Human-owned gates (cannot be closed in this environment)

Per last sync's scope honesty: **paired-phone validation** + **real Chatwoot round-trip** are
human-owned gates **before** merging to `origin/main`. Code-complete + all `go test ./...`
green + contract-drift clean = PR-ready, **not** merge-ready. No prod cutover here.

## 7. Commands already run (orientation — safe/read-only)

```
git fetch origin --prune --tags                              # ok
git fetch upstream --prune "refs/heads/*:refs/remotes/upstream/*"   # branches (tag clobber avoided)
git fetch upstream "refs/tags/*:refs/tags/upstream/*"        # upstream tags namespaced
```

Nothing pushed, no branches created, no working-tree changes.
