# 04-plan consolidation rationale

Two parallel P-stage drafts produced:
- `04-plan-claude.md` (244 lines, 8 slices, ~7.75d, written by Claude/general-purpose agent)
- `04-plan-codex.md` (238 lines, 7 slices, 7.5-9.0d, written by Codex via codex-rescue)

The final `04-plan.md` is a deliberate merge. This doc records the divergence calls so a code-owner reviewing the consolidated plan can see which source each decision came from and why.

## Divergence calls

| # | Topic | Claude | Codex | Final | Why |
|---|---|---|---|---|---|
| 1 | Preflight vs reset bundling | Slice 0 = preflight, Slice 1 = reset+smoke | Slice 0 = preflight+reset+release-rail bundled | **Claude's split, plus Codex's release-rail folded into Slice 1** | Smoke must precede release-rail reapply so a `go test` failure on upstream-vanilla is diagnosable as "upstream bug" not "Helm/CI YAML conflict." Slice 1 internal ordering: reset → smoke → release rail. CI workflows live for every later slice's PR review. |
| 2 | `is_from_me` consumer logic placement | Slice 4 (lockstep cutover) | Slice 5 (taxonomy audit) | **Split: behavior in Slice 4, docs in Slice 5** | Lockstep ruled out dual-listen. The chatwoot-app filter switch from env-knob to payload-field happens the moment cutover lands. Docs (deprecation note, `### Common Payload Fields`) belong in the taxonomy slice. |
| 3 | Fork sweep position | Slice 6 (after chatwoot) | Slice 3 (before chatwoot) | **Claude's post-chatwoot order + mandatory chatwoot regression check at Slice 6 end** | Highest-risk Slice 4 ships on the cleanest possible tree. Fork-only files (info-cache, proxy, audio) don't overlap chatwoot. The real risk Codex identified is fork-side `device_manager.go` shadowing chatwoot's `DeviceInstance` assumptions; the Slice 6 mandatory chatwoot round-trip test catches it before release. |
| 4 | OQ8 `device_manager.go` handling | Own commit in Slice 6 | Bundled into Slice 3 sweep | **Codex's bundle-by-default + Claude's hazard flagging via explicit inspect-first step** | `git log v8.1.2..upstream/main -- device_manager.go` runs at Slice 6 start. If upstream's 2 commits don't overlap fork-edited functions: bundle. If they do: own commit, +0.5d. The Slice 6 chatwoot regression check is the safety net either way. |
| 5 | Caller-site count | "39 sites per local grep" | "S-listed 45 usecase call sites" | **"~40 callers; audit fresh at slice start"** | Neither draft cited grep output directly. Hedge until the rg audit runs. |
| 6 | Slice count | 8 (0-7) | 7 (0-6) | **8 (0-7)** | Claude's split between reset (Slice 1) and release cut (Slice 7) gives clearer checkpoint boundaries. Codex's compaction is appealing but bundles "release infrastructure reapply" with "release cut" — different work, different risk profiles. |
| 7 | Effort estimate | 7.75d | 7.5-9.0d | **8.25d (range 8.25-8.75)** | Reset+release-rail bundling adds 0.25d to Slice 1 net (vs splitting into 0.5d each). OQ8 inspection adds +0.5d contingency. Within both source ranges. |

## Areas where both agreed (no divergence)

- Reset+reapply strategy with `pre-upgrade-snapshot-2026-05-14` rollback anchor.
- **OQ3 → new file `forward_history_sync.go`** — both drafts independently arrived at the same call. Codex's reasoning: easier to audit, easier to upstream. Claude's reasoning: cleaner reapply diff than a hunk inside upstream's 484-line dispatcher.
- Slice 4 is the ship-stoppable lockstep gate; staging rollback rehearsal mandatory.
- `is_from_me` is the consumer-side signal; `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` is dead-knob-still-documented.
- Total budget ~8 person-days.
- Vertical-slicing invariant: each slice has a paired-phone-scenario checkpoint.

## Calls deferred to slice execution

- **Slice 2 caller count** — audit via `rg -n 'utils\.ValidateAndNormalizeJID\(' src/usecase/` at slice start.
- **Slice 6 `device_manager.go` overlap** — `git log v8.1.2..upstream/main -- device_manager.go` at slice start. If function-overlap with fork edits: own commit. If disjoint: bundle.
- **Slice 4 chatwoot-app PR aging window** — sync points between teams advised if Slice 4 cutover slips beyond ~10 days from chatwoot-app PR draft.

## Why this consolidation pattern

Two parallel drafts cost ~ extra agent-runtime in exchange for two independent stresses on the design surface. Where both drafts agreed, the call is robust. Where they diverged, the divergence itself surfaces the tradeoff. The final plan picks the side that's stronger under QRSPI invariants (checkpointability, vertical-slicing, rollback discipline), with the loser's concern preserved as a guardrail (the Slice 6 chatwoot regression check is Codex's concern operationalized despite Claude's order winning).

If a future P-stage workstream wants to repeat this pattern: parallel runs are most valuable when the divergence space includes ordering decisions and slice-boundary calls. They're less valuable when the underlying contract (D + S) is fully constrained — at that point a single-agent draft suffices and the second run mostly produces the same plan with cosmetic variation.
