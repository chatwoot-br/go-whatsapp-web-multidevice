# Workstream: sync chatwoot-br fork with upstream v8.5.0

Started: 2026-05-14 · Methodology: [QRSPI](../../../../workspaces/llm-wiki/.queries/2026-05-11-qrspi.md)

## Context

- Fork `main` is at `v8.1.2+1` (2026-01-26).
- Upstream `aldinokemal/go-whatsapp-web-multidevice` is at `v8.5.0`, **89 commits ahead**.
- Critical upstream addition: native Chatwoot integration (`44a128c` + follow-ups) — paradigm shift for the fork's reason-to-exist.
- Other overlaps: `src/pkg/utils/phone.go` (upstream) vs fork's `ValidateAndNormalizeJID`; native LID resolution vs fork's `MergeLIDChat`; native webhook caption inclusion vs fork's v8.1.0+3 fix.
- Baseline impact assessment lives in `~/workspaces/llm-wiki/.queries/2026-05-14-chatwoot-gowa-integration.md`.

## QRSPI stage map

| Stage | File | Status |
|---|---|---|
| **Q** Questions | `00-questions.md` | ✓ done 2026-05-14 |
| **R** Research (ticket hidden) | `01-research.md` | ✓ done 2026-05-14 |
| **D** Design (~200 lines) | `02-design.md` | ✓ done 2026-05-14 (91 lines, decision-dense) |
| **S** Structure (signatures/types) | `03-structure.md` | **next** — after D review |
| **P** Plan (vertical slices) | `04-plan.md` | drafted (revisit after D) |
| **W** Worktree | `05-worktree.md` | pending |
| **I** Implement | (in worktree) | pending |
| **PR** Pull Request | `06-pr.md` | pending |

## Q-stage answers (summary)

- **Q1 Upgrade strategy** → C: Reset + re-apply deltas
- **Q2 Chatwoot integration** → A: Adopt + migrate chatwoot-app
- **Q3 Phone normalization** → C: Layer fork's BR rules on upstream's
- **Q4 Scope** → A: All-in

## Invariants

- Each stage is a **separate context window** (no mega-prompt).
- **Disk-resident artifacts**: every stage produces a file here before moving on.
- **Don't outsource the thinking**: human picks Q answers, reviews D before P, reviews P before W.
- **2026 framing**: 2–3× speed-up with near-human quality, not 10× slop.

## How to drive this

1. Answer Q1/Q2/Q3 in `00-questions.md` — record decisions there.
2. Open a fresh chat for **R** with `01-research.md` as the destination and the *ticket text hidden* — produce facts only.
3. Open a fresh chat for **D** with research + answers → produce ~200-line design doc.
4. Review D as code-owner before any plan exists.
5. **S** then **P**; P slices vertically (mock end-to-end first, integrate last).
6. **W**: `git worktree add` per slice; review the plan once, spot-check after.
7. **I**: implement per slice.
8. **PR**: deep review against code, not plan.
