# Execution — team orchestration

Agent-team execution of `04-plan.md`. The plan defines what each slice does; this doc defines who does what when, in which worktree.

## Scope honesty

- **In scope**: all Go code changes per the plan, `go test ./...` validation, new unit + integration tests, mocked WhatsApp client + Chatwoot HTTP API for e2e-style coverage. Tag creation (local; not pushed).
- **Out of scope** (require human + hardware/infra not present this session): paired-phone validation, real Chatwoot instance round-trip, cross-repo chatwoot-app Rails changes, staging deploy, prod cutover.
- **PR-ready state**: code-complete with all Go tests green; fork-specific features have unit/integration test coverage with mocked external dependencies. Paired-phone validation is the human-owned gate before merging to chatwoot-br/main.

## Phase plan

| Phase | Agents | Slices | Mode | Output |
|---|---|---|---|---|
| 1 | 1 | 0, 1 | serial on trunk | `upgrade/v8.5.0-sync` branch with preflight + reset + release rail commits |
| 2 | 3 | 2, 3, 6 | parallel worktrees | three slice branches with implementations |
| 3 | 1 | merge | serial on trunk | upgrade branch with 2+3+6 integrated; conflicts resolved |
| 4 | 1 | 4 | serial on trunk | chatwoot env wiring + Helm values |
| 5 | 1 | 5 | serial on trunk | webhook taxonomy docs + env audit |
| 6 | 1 | 7 | serial on trunk | CHANGELOG + version bump + LOCAL tag (no push) |
| 7 | 1 | e2e tests | parallel worktree | new test files for fork-specific features |
| 8 | 1 | validation | serial on trunk | final go test ./... + PR-readiness summary |

## Worktree map

| Phase | Worktree path | Branch | Lifetime |
|---|---|---|---|
| 1 | none (main checkout) | `upgrade/v8.5.0-sync` (NEW) | permanent (lives until PR merged) |
| 2 — Slice 2 | `../gowa-wt-s2-brphone` | `slice2-br-phone` | merged + removed in Phase 3 |
| 2 — Slice 3 | `../gowa-wt-s3-lid-history` | `slice3-lid-history` | merged + removed in Phase 3 |
| 2 — Slice 6 | `../gowa-wt-s6-fork-sweep` | `slice6-fork-sweep` | merged + removed in Phase 3 |
| 7 | `../gowa-wt-e2e-tests` | `slice-e2e-tests` | merged + removed in Phase 8 |

## Conflict pre-emption

Predicted conflict zones when merging Slices 2+3+6:
- `src/.env.example` — Slice 3 adds `history_sync_complete` to `WHATSAPP_WEBHOOK_EVENTS`; Slice 6 may add proxy env vars. Append-only merges should be clean.
- `src/config/settings.go` — Slice 6 adds proxy settings + cache TTL; Slice 3 doesn't touch this file. Likely clean.
- `docs/webhook-payload.md` — only Slice 3 touches it in Phase 2 (Slice 4 + Slice 5 update later). Clean.
- `readme.md` — Slice 3 + Slice 6 may both add doc bullets. Likely clean (different sections).
- `CHANGELOG.md` — every slice appends its own entry. Order matters but conflicts are mechanical.

**Conflict resolution strategy**: merge in order Slice 3 → Slice 2 → Slice 6 (interface-first → caller-sweep → infrastructure). Merge agent has authority to resolve mechanical conflicts; flags semantic conflicts back for review.

## Phase exit criteria

Each phase produces a commit on its target branch + a status line in this doc. Subsequent phases gate on prior phases' exit:

- Phase 1 exit: `go test ./...` green on upstream-vanilla; Helm lints; CI workflow dry-run accepts `v8.5.0+1` shape.
- Phase 2 exit: each of slice2/slice3/slice6 branches has `go test ./...` green on their isolated worktree.
- Phase 3 exit: merged upgrade branch has `go test ./...` green; no merge conflicts unresolved.
- Phase 4 exit: chatwoot env vars wired; `go test ./...` still green.
- Phase 5 exit: docs updated; final taxonomy audit grep returns expected hits.
- Phase 6 exit: CHANGELOG complete; AppVersion bumped; local tag `v8.5.0+1` created (not pushed).
- Phase 7 exit: new test files committed; `go test ./...` green including new tests; coverage of fork-specific features (BR phone, LID dedup, history_sync_complete, hmac signing) demonstrably present.
- Phase 8 exit: final summary written; PR description draft ready.

## Tracking

Phase status updated below as agents complete:

- [x] Phase 1 — Slice 0 + Slice 1
- [x] Phase 2 — Slices 2, 3, 6 (parallel)
- [x] Phase 3 — Merge 2+3+6
- [x] Phase 4 — Slice 4
- [x] Phase 5 — Slice 5
- [x] Phase 6 — Slice 7 release
- [x] Phase 7 — E2E tests
- [x] Phase 8 — Final validation
