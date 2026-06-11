# Upstream Sync Runbook (resync the fork onto upstream)

How to periodically resync this fork (`chatwoot-br/go-whatsapp-web-multidevice`, the
`origin` product) with the upstream project (`aldinokemal/go-whatsapp-web-multidevice`,
the `upstream` product), reproducibly.

> **"Rebase" vs merge.** This is colloquially called "rebasing onto upstream," but the
> mechanism is a **merge**, not `git rebase`. The fork ships from `origin/main` with a
> linear release history and the house rule is **no `--amend` / no force-push** (revisions
> are new commits; squash happens at PR merge). Rebasing the fork's 40+ commits would
> rewrite SHAs, force a push, and create one conflict point per commit. A merge resolves
> the divergence **once** and preserves both histories. Use merge.

This is the durable template. Each actual run gets its own dated working directory under
`.workstreams/<date>-upstream-v<X.Y>-sync/` (see [§9](#9-per-run-bookkeeping)). Prior runs:
`.workstreams/2026-05-14-upstream-v8.5-sync/` (v7.8→v8.5, the big one),
`.workstreams/2026-06-11-upstream-v8.7-sync/` (v8.5→v8.7).

---

## 1. Invariants — the fork's strategy (do not silently change these)

Internalize before touching code. These are settled decisions, not open questions:

1. **The fork's own Chatwoot integration is canonical; GoWA-native chatwoot stays dormant.**
   Upstream has its *own*, independently-evolving `infrastructure/chatwoot/` (now larger than
   the fork's — `pgimport/`, `provision.go`, `ui/rest/chatwoot.go`, `pkg/utils/chatwoot.go`).
   The fork settled in the v8.5 sync ("Q2 reversal") **not** to adopt it: native module stays
   in-tree but dormant, `CHATWOOT_ENABLED` defaults **false**. Do not let an upstream chatwoot
   commit re-route the live path. (`.workstreams/2026-05-14-upstream-v8.5-sync/10-q2-reversal.md`)
2. **The integration contract with `chatwoot-app` is the *standard* webhook output**
   (`forwardToWebhooks`, not `forwardToChatwoot`). What must stay stable is the payload shape
   that `chatwoot-app`'s `Webhooks::WhatsappWebController` parses, including the fork-custom
   `history_sync_complete` event, the `chat_name`/`sender_name` fields, and HMAC
   `X-Hub-Signature-256` signing. Any upstream change to webhook payloads → re-run the
   **contract-drift check** ([§7](#7-the-chatwoot--webhook-gate)).
3. **Fork features to preserve through every merge** (the reason the fork exists):
   - BR phone normalization — `src/pkg/utils/phone_br.go` + caller sweep
   - LID dedup + `history_sync_complete` dispatch — `infrastructure/whatsapp/history_sync.go`
   - Full history sync + `ON_DEMAND` capability
   - Short-term info cache — `infrastructure/whatsapp/info_cache.go`, `pkg/cache/`
   - SOCKS5/HTTP/HTTPS proxy support for WhatsApp connections
   - Webhook taxonomy: `chat_name`/`sender_name` payload fields + HMAC signing
   - Startup resilience: bounded `InitWaDB` retry, ambiguous-`IsOnWhatsApp` fall-through
4. **Release rail: `vX.Y.Z+N`.** The fork rebases its release version onto the upstream base
   tag and appends `+N` (e.g. `v8.5.0+5`). `AppVersion` lives in `src/config/settings.go`;
   changes are logged in `CHANGELOG.md` (Keep a Changelog). A pushed `v*` tag triggers
   `release.yml`. This `+N` scheme **never collides** with upstream's clean `vX.Y.Z`.

---

## 2. Standing hazards (re-check every run)

| Hazard | Why it bites | Mitigation |
|---|---|---|
| **Tag-name collision** | Legacy fork tags `v7.8.0`/`v7.8.2` point to *fork* commits; upstream's same-named tags point elsewhere. `git fetch upstream --tags` fails: *"would clobber existing tag."* | Fetch upstream tags into a **namespace** (§4). Never force-overwrite fork tags. Never push an upstream tag to `origin`. |
| **whatsmeow API drift** | The fork's event + history-sync layer calls deep into `go.mau.fi/whatsmeow`, which upstream bumps almost every release. **This is the real risk — not text conflicts.** Code can merge cleanly and still fail to compile/behave against the newer whatsmeow. | `go build ./... && go vet ./...` against the new whatsmeow is the **first** gate, before any feature work. |
| **Chatwoot / webhook reconciliation** | Upstream evolves its own chatwoot code and occasionally changes webhook payload shape. | Partition it out (§6); keep native dormant; re-run contract-drift (§7). |
| **Production push** | Pushing `gitops`/release tags auto-deploys; force-push is forbidden. | PR to `origin/main`; tag/push is a human-gated production action (§8). |

---

## 3. Step 0 — Pre-flight assessment (decide the shape of this run)

Run these read-only commands, record the answers in the run's `00-plan.md`:

```bash
cd src && git -C .. remote -v          # confirm origin=chatwoot-br, upstream=aldinokemal
MB=$(git merge-base origin/main upstream/main)        # current branch point
git rev-list --left-right --count origin/main...upstream/main   # fork-ahead / upstream-ahead
git log -1 --format='%h %ci %s' "$MB"                 # where the fork last synced
git tag -l 'upstream/v*' --sort=-v:refname | head     # latest upstream releases
```

**Decisions to make:**

- **Target tag, not `upstream/main`.** Pick the latest *stable* upstream tag
  (`upstream/v<X.Y.Z>`). `upstream/main` is a moving target and usually carries unreleased,
  higher-risk commits. Targeting a tag makes the run reproducible.
- **Partition: core vs tail.** List `git log upstream/v<TARGET>..upstream/main` and check
  where chatwoot/webhook commits fall. In practice they cluster in the *unreleased tail*.
  If so, the target-tag sync is **chatwoot-free** (Phase A) and the tail is a **separate,
  later** Phase B (§6, §7). Verify per run — don't assume.
- **Conflict surface** (compute against the **target tag**, not `upstream/main`):

  ```bash
  comm -12 <(git diff --name-only "$MB" upstream/v<TARGET> | sort) \
           <(git diff --name-only "$MB" origin/main        | sort)
  ```
  Files both sides changed = where conflicts will land. Note the highest-churn fork file
  (historically `infrastructure/whatsapp/history_sync.go`).

---

## 4. Step 1 — Update remotes (idempotent, read-only)

```bash
git fetch origin --prune --tags
git fetch upstream --prune "refs/heads/*:refs/remotes/upstream/*"   # branches only
git fetch upstream "refs/tags/*:refs/tags/upstream/*"               # tags → namespace, no clobber
```

> Do **not** run `git fetch upstream --tags` — it clobbers on the legacy `v7.8.x` names.
> Reference upstream releases as `upstream/v<X.Y.Z>` throughout.
> `origin/upstream` is a stale mirror branch — ignore it; track `upstream/main`.

---

## 5. Step 2 — Merge, resolve, validate (Phase A: target tag)

```bash
git switch -c upgrade/v<TARGET>-sync origin/main      # new branch off the fork tip
git merge upstream/v<TARGET>                          # expect conflicts in the §3 surface
```

**Resolve in dependency order** (interface-first → callers → infrastructure):

1. `pkg/utils/` (JID, phone, general) — fork's BR-phone + LID primitives
2. `domains/` interfaces + `infrastructure/chatstorage/` (repo + migrations — **append-only**)
3. `infrastructure/whatsapp/` event layer (`event_message*.go`, `device_manager.go`, `init.go`)
4. **`infrastructure/whatsapp/history_sync.go`** — highest-churn; re-apply fork's full-sync /
   LID-dedup / `history_sync_complete` over upstream's edits **and** the new whatsmeow API
5. `usecase/` (`chat.go`, `send.go`)
6. `config/settings.go`, `cmd/`, `.env.example` — usually append-only (proxy/cache/webhook vars)
7. mechanical: `readme.md`, `docs/webhook-payload.md`, `.github/workflows/*`, `go.mod`/`go.sum`

Preserve every fork feature in [§1.3](#1-invariants--the-forks-strategy-do-not-silently-change-these).
Flag any *semantic* conflict (behavior change, not text) for human review rather than guessing.

**Validation gates — in order, each must pass before the next:**

```bash
cd src && go mod tidy        # reconcile deps: new modules (e.g. modernc.org/sqlite), bumps, go ver
cd src && go build ./...     # GATE 1 — compiles against new whatsmeow (the load-bearing gate)
cd src && go vet ./...       # GATE 2 — static analysis
cd src && go test ./...      # GATE 3 — all tests green (fix fork tests broken by upstream changes)
```

---

## 6. Phase B — the unreleased tail (optional, separate, later)

Only **after** Phase A lands. This is where chatwoot reconciliation + webhook-shape changes live.

```bash
git switch -c upgrade/v<TARGET>-tail-sync origin/main   # after Phase A merged to origin/main
git merge upstream/main                                 # the commits past the target tag
```

- Reconcile upstream's chatwoot edits against the fork's `infrastructure/chatwoot/`. Keep the
  fork integration canonical; **native stays dormant** (`CHATWOOT_ENABLED=false`). Decide
  per-file whether to absorb upstream additions or leave them dormant — never let them
  silently take over the live webhook path.
- **Run the contract-drift check (§7) — mandatory** whenever the tail touches webhook payloads.
- Same build/vet/test gates as Phase A; same release rail.

---

## 7. The chatwoot / webhook gate (contract-drift check)

The load-bearing chatwoot gate. Whenever a merged commit changes GoWA's **standard** webhook
output, verify it still matches what `chatwoot-app`'s fork-owned controller parses.

- Compare GoWA `forwardToWebhooks` output against
  `chatwoot-app:fork/app/.../webhooks/whatsapp_web_controller.rb` +
  `incoming_message_whatsapp_web_service.rb` (top-level keys `device_id`/`event`/`payload`,
  the `event`-value switch, nested payload fields, HMAC `X-Hub-Signature-256` over raw body).
- Template + worked example: `.workstreams/2026-05-14-upstream-v8.5-sync/11-contract-drift.md`.
- Output: a short doc stating *N breaking / N behavioral / HMAC stable*. Non-clean → fix in
  the fork (preserve the contract) before the PR is merge-ready.

---

## 8. Gates: automatable vs human-owned

**PR-ready** (can finish in-session) = code-complete, conflicts resolved, `go build`+`vet`+`test`
green, contract-drift clean, CHANGELOG + `AppVersion` bumped, **local** `v<TARGET>+N` tag created.

**Merge-ready / shippable** (human-owned, cannot close here):

- **Paired-phone validation** against a real WhatsApp device.
- **Real Chatwoot round-trip** (send/receive through a live instance).
- **Push** of the branch and the `v<TARGET>+N` tag (pushing `v*` triggers `release.yml` → build
  → image). Pushing is a **production action** — hand the exact command to the user; do not push
  unprompted or force-push.

Release rail steps (do locally, push is human-gated):

```bash
# bump AppVersion in src/config/settings.go to "v<TARGET>+N"; add CHANGELOG entry
git tag v<TARGET>+N           # local only
# user pushes branch + tag when validation passes
```

---

## 9. Per-run bookkeeping

Each run creates `.workstreams/<YYYY-MM-DD>-upstream-v<X.Y>-sync/` containing at least:

- `00-plan.md` — pre-flight findings (§3): divergence counts, merge-base, target tag,
  conflict surface, partition decision, dep deltas (esp. whatsmeow).
- `logs/` — captured `go build` / `go vet` / `go test` output per gate.
- contract-drift note (if Phase B / webhook changes), modeled on the v8.5 run's `11-`.

Keep dated runs as the audit trail; this `docs/upstream-sync.md` stays the reusable template.

---

## 10. Quick reference (copy-paste, set `TARGET`/`N`)

```bash
TARGET=8.7.0 ; N=1                                # <-- set per run

# 0. fetch (idempotent, safe)
git fetch origin --prune --tags
git fetch upstream --prune "refs/heads/*:refs/remotes/upstream/*"
git fetch upstream "refs/tags/*:refs/tags/upstream/*"

# 1. assess
MB=$(git merge-base origin/main upstream/main)
git rev-list --left-right --count origin/main...upstream/main
git log --oneline "upstream/v$TARGET..upstream/main"          # the tail (chatwoot/webhook?)
comm -12 <(git diff --name-only "$MB" "upstream/v$TARGET"|sort) \
         <(git diff --name-only "$MB" origin/main|sort)       # conflict surface

# 2. merge core (Phase A)
git switch -c "upgrade/v$TARGET-sync" origin/main
git merge "upstream/v$TARGET"
# ...resolve (§5 order)...
cd src && go mod tidy && go build ./... && go vet ./... && go test ./...

# 3. release rail (local), PR to origin/main; push + paired-phone = human gate
```

## See also

- `CLAUDE.md` — build/test commands, architecture, anti-patterns
- `docs/chatwoot.md` — chatwoot integration reference
- `docs/webhook-payload.md` — webhook payload contract
- `.workstreams/2026-05-14-upstream-v8.5-sync/` — full worked example (incl. contract-drift)
