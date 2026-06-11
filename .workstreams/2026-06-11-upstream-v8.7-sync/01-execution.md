# 01 — Execution tracker (v8.5.0+5 → upstream/main)

Branch: `upgrade/v8.7.0-sync` (off `origin/main` `3cf8e68`). Plan: `00-plan.md` +
`/Users/woot/.claude/plans/cozy-discovering-parasol.md`. Method: **merge** (no rebase).

Anchors: merge-base `17af98e`; target tag `upstream/v8.7.0` (`cf8b1b4`);
tail `upstream/v8.7.0..upstream/main` = 12 commits (`131b99b` tip).

## Phase status

- [x] **Phase 0 — branch + scaffold + baseline.** `go build/vet/test` GREEN on `3cf8e68`
      (pre-merge). BUILD_EXIT=0 VET_EXIT=0 TEST_EXIT=0. Log: `logs/phase0-baseline.log`.
      go 1.26.4. e2e package present behind `//go:build e2e` (`src/e2e/integration_test.go`).
- [x] **Phase A — merge upstream/v8.7.0** (chatwoot-free core). Commit `0824c9e`.
      13 conflicts resolved + 4 post-merge wiring fixes. **No whatsmeow API drift** —
      only breakage was auto-merged upstream code referencing the fork-dropped
      `NormalizeJIDFromLID` (rewired to `NormalizeJIDFromLIDWithContext` in
      `event_label.go` + `sqlite_repository.go`). 1 test fix (`chat_test.go` `GetChat`
      stub for fork's sender-name lookup). Adopted upstream reactions / secret-edit
      decrypt / pure-Go sqlite / label webhooks / presence-pulse / send-retry. Fork
      `CLAUDE.md` restored (upstream deleted it for AGENTS.md; AGENTS.md kept additively).
      build/vet/test + e2e GREEN — `logs/phaseA-build-vet.log`, `logs/phaseA-tests.log`.
- [x] **Phase B — merge upstream/main tail** (chatwoot reconcile + session_id). Commit `42cba0d`.
      Base advanced to **upstream v8.8.0** (upstream/main moved past v8.7.0 during the sync).
      7 conflicts + chatwoot reconciliation. **Decisions (`02-contract-drift.md`):**
      (1) custom attr `waha_whatsapp_jid`→`gowa_whatsapp_jid` — write gowa, **read both**
      (waha fallback in `client.go` + inbound `ui/rest/chatwoot.go`) so pre-existing
      contacts keep routing; cross-repo grep clean. (2) Adopt upstream preserve-existing-
      1:1-name; dropped fork's overwrite test, kept upstream's 3. (3) `session_id` =
      additive top-level webhook key (chatwoot-app ignores unknowns); HMAC + fork fields
      intact. build/vet/test + e2e GREEN — `logs/phaseB-tests.log`. **0 breaking changes
      to the chatwoot-app contract.**
- [x] **Phase R — release rail.** Commit `64acbd3`. AppVersion → `v8.8.0+1`
      (`settings.go`), CHANGELOG entry, **local tag `v8.8.0+1`** (matches validate-tag
      regex `^v[0-9]+\.[0-9]+\.[0-9]+\+[0-9]+$`). Not pushed.
- [x] **Phase V — validation + PR.** Final `build/vet/test + e2e` GREEN
      (`logs/phaseV-final.log`). Coverage: chatwoot 61%, pgimport 78%, chatstorage 43%,
      utils 47%, whatsapp 34%. PR draft: `03-pr-description.md`. Untestable/UAT checklist:
      `untestable-surfaces.md`. **Branch push + PR-open + paired-phone UAT are human-owned.**

## Result

Code-complete & PR-ready on `upgrade/v8.7.0-sync`. 5 commits over `origin/main`:
phase-0 scaffold, phase-A merge (`0824c9e`), phase-A tracker, phase-B merge (`42cba0d`),
release `v8.8.0+1` (`64acbd3`) + this bookkeeping. Nothing pushed. whatsmeow drift was a
non-event; the real work was chatwoot reconciliation (`02-contract-drift.md`).

## Decisions / notes (append as they happen)

- Native chatwoot stays dormant (`CHATWOOT_ENABLED=false`); fork integration canonical (v8.5 Q2 reversal).
- e2e gate: `go test -tags=e2e ./...`.
