# P — Plan (vertical slices)

QRSPI P stage. Produced from S. **Sliced vertically** — each slice is end-to-end with a checkpoint where you can stop and ship.

> Models default to horizontal ("all DB → all services → all API → all frontend"). Vertical surfaces integration failures early.

## Slice 0 — Preflight (no code changes)

- [ ] `git fetch upstream` and confirm `upstream/main` HEAD matches the SHA in `01-research.md`.
- [ ] `git checkout -b upgrade/v8.5.0-sync` from current fork `main`.
- [ ] Tag current state as `pre-upgrade-snapshot-2026-05-14` for rollback.
- [ ] Verify all fork-specific tests pass on current main (baseline gate).

**Checkpoint:** Baseline green; rollback tag exists.

---

## Slice 1 — Whatsmeow + protocol updates (smallest, highest-value)

- [ ] Cherry-pick / merge the ~20 `chore: update whatsmeow to latest` commits up to the last one before any contentious feature commit.
- [ ] Resolve conflicts in `go.mod` / `go.sum`.
- [ ] Run integration tests against paired phone.

**Checkpoint:** Gateway connects, history syncs, messages send/receive. *Ship-stoppable here.*

---

## Slice 2 — Healthcheck + ops fixes

- [ ] Land `fc7fe7b` healthcheck endpoint.
- [ ] Land `75869b5` Docker permission fix.
- [ ] Update Helm chart to wire the new healthcheck probe.

**Checkpoint:** k8s liveness probe green.

---

## Slice 3 — LID + phone reconciliation (per Q3 answer)

- [ ] Apply Q3-decision steps to `phone.go` / `general.go`.
- [ ] Land upstream LID resolution (`17ff32f`, `d718ef8`, `40b0875`).
- [ ] Verify fork's `MergeLIDChat` is either retired or still needed.
- [ ] BR phone normalization regression test.

**Checkpoint:** Paired phone with BR + LID account: contacts non-duplicated; messages routed correctly.

---

## Slice 4 — Chatwoot integration adoption (per Q2 answer)

_(Skip this slice entirely if Q2 = B.)_

- [ ] Land `44a128c` + `909b6e6` + `29907ee` + `3b87f4e` + `2d27ea8`.
- [ ] Configure new env vars from `03-structure.md`.
- [ ] If Q2 = A: refactor chatwoot-app (Rails) consumer to use new endpoints; coordinate cutover.
- [ ] If Q2 = C: drop `ui/rest/chatwoot.go`; keep `infrastructure/chatwoot/`.

**Checkpoint:** End-to-end message round-trip via chatwoot-app inbox.

---

## Slice 5 — Webhook event taxonomy expansion

- [ ] Land `437df12` phone in contact payload.
- [ ] Land `c428afa` typing events.
- [ ] Land `a8b5ed8` calls in chatstorage + `381c381` contacts in chatstorage.
- [ ] Land `00ee65b` ContactsArrayMessage.
- [ ] Land `306391e` captions in payload (verify against fork's v8.1.0+3).
- [ ] Update chatwoot-app dedup logic for new event types.

**Checkpoint:** All event types reachable from a controlled phone action; chatwoot-app handles them.

---

## Slice 6 — Feature commits (per Q4 answer)

_(Skip entirely if Q4 = B or C.)_

- [ ] GIF playback (`a6f7b44`, `a6b6a02`, `d32aadf`).
- [ ] CTWA Meta Ads (`fe7d2c7`).
- [ ] Document thumbnails (`ea27ef2`, `8606995`).
- [ ] Ghost mentions (already in upstream — verify it's pulled in via earlier slices).
- [ ] Archived chats filter (`e13966a`).

**Checkpoint:** Each feature reachable through REST + UI.

---

## Slice 7 — Release

- [ ] CHANGELOG.md entry: `[v8.5.0+1]` with Upstream Changes / Fork Changes sections.
- [ ] Version bump in `src/cmd/root.go` (or wherever).
- [ ] Tag, push, trigger goreleaser.
- [ ] Helm chart bump.

**Checkpoint:** New image published; chatwoot-app deployment pin updated.

---

**Vertical-slicing invariant:**

- Each slice is shippable in isolation. If Slice 4 blows up, Slices 0–3 still ship as `v8.5.0+1-partial`.
- Each slice has its own worktree (W stage).
- Plan review is a spot-check, not a deep audit — the deep review happens against code in PR.
