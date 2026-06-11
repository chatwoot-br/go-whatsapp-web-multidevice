# 01 — Execution tracker (v8.5.0+5 → upstream/main)

Branch: `upgrade/v8.7.0-sync` (off `origin/main` `3cf8e68`). Plan: `00-plan.md` +
`/Users/woot/.claude/plans/cozy-discovering-parasol.md`. Method: **merge** (no rebase).

Anchors: merge-base `17af98e`; target tag `upstream/v8.7.0` (`cf8b1b4`);
tail `upstream/v8.7.0..upstream/main` = 12 commits (`131b99b` tip).

## Phase status

- [x] **Phase 0 — branch + scaffold + baseline.** `go build/vet/test` GREEN on `3cf8e68`
      (pre-merge). BUILD_EXIT=0 VET_EXIT=0 TEST_EXIT=0. Log: `logs/phase0-baseline.log`.
      go 1.26.4. e2e package present behind `//go:build e2e` (`src/e2e/integration_test.go`).
- [ ] **Phase A — merge upstream/v8.7.0** (chatwoot-free core, 28-file surface).
- [ ] **Phase B — merge upstream/main tail** (chatwoot reconcile + session_id + contract-drift).
- [ ] **Phase R — release rail** (`v8.7.0+1`).
- [ ] **Phase V — validation + PR.**

## Decisions / notes (append as they happen)

- Native chatwoot stays dormant (`CHATWOOT_ENABLED=false`); fork integration canonical (v8.5 Q2 reversal).
- e2e gate: `go test -tags=e2e ./...`.
