# W — Worktree

QRSPI W stage. Dex's keynote framing is "one worktree per slice — parallelisable, clean rollback." That's right under independent slices. Under reset+reapply (this workstream's Q1=C), most slices serially depend on the previous slice's commits being on trunk, so per-slice worktrees mostly add ceremony without enabling real parallelism. **Worktrees earn their keep at two specific points** — call them out instead of pretending the literal "one-per-slice" template fits.

## Default: single-trunk operation

All 8 slices commit to `upgrade/v8.5.0-sync` in the main checkout. Rollback is uniform across slices: `git reset --hard pre-upgrade-snapshot-2026-05-14` + redeploy prior image.

```bash
git switch upgrade/v8.5.0-sync     # from main checkout
# implement slice N
git commit -m "..."
# next slice continues on same branch
```

## Where worktrees earn their keep

### Mandatory: Slice 4 staging worktree

Lockstep cutover (Slice 4) needs an isolated staging deploy target. The main checkout must stay usable for hotfix work on the previous prod image during the cutover window. A worktree pointed at `upgrade/v8.5.0-sync` gives staging a clean tree without blocking the main checkout.

```bash
git worktree add ../gowa-wt-slice4-staging upgrade/v8.5.0-sync
cd ../gowa-wt-slice4-staging
# point staging Docker build here; populate CHATWOOT_* env
# run the 5 paired-phone scenarios from the Slice 4 checkpoint
# rehearse rollback (deploy → revert → redeploy) before prod cutover
```

Cleanup after Slice 4 ships:

```bash
git worktree remove ../gowa-wt-slice4-staging
```

### Optional: parallel Slice 2 + Slice 6 (two-engineer mode)

Per the consolidated plan effort table: *"Slices 2 and 6 parallelize against each other after Slice 3 completes."* If a second engineer picks up Slice 6 (fork-only delta sweep) while the primary finishes Slice 2 (BR phone caller sweep):

```bash
# Engineer A continues on main checkout (Slice 2 work)
# Engineer B opens parallel worktree:
git worktree add ../gowa-wt-slice6-sweep upgrade/v8.5.0-sync
cd ../gowa-wt-slice6-sweep
# implement Slice 6 commits (audio/PTT, proxy, info-cache, S3 ext, device_manager.go)
```

Slice 2 touches `src/usecase/{send,group,message,chat,user,newsletter}.go`; Slice 6 touches `src/infrastructure/whatsapp/{info_cache,device_manager,client_lifecycle}.go` + `src/pkg/utils/general.go` + Docker/Helm. The slice ordering was designed so these surfaces don't collide. Merge back to `upgrade/v8.5.0-sync` linearly after each slice's checkpoint; conflicts would surface only if scope crept.

### Discouraged: per-slice worktree for Slices 0, 1, 3, 5, 7

These slices either don't produce code commits (Slice 0 is tag+branch only), depend on the previous slice's tree being on trunk (Slices 3, 5, 7), or are large-surface single-engineer work where a worktree is overhead (Slice 1's release rail reapply). Just work on trunk.

## Slice-to-worktree mapping

| Slice | Worktree | Justification |
|---|---|---|
| 0 Preflight | none | Tag + branch on main checkout. |
| 1 Reset + smoke + release rail | none | Single-engineer sequential work on trunk. |
| 2 BR phone + ~40 callers + tests | optional | Pair-mode with Slice 6 after Slice 3 completes. |
| 3 LID dedup + history_sync_complete | none | Storage interface changes; serial dependency for Slice 4. |
| 4 Chatwoot lockstep cutover | **mandatory** | Staging deploy needs isolation; main checkout stays available for hotfix. |
| 5 Webhook taxonomy + env audit | none | Docs + verification on trunk. |
| 6 Fork sweep + OQ8 `device_manager.go` | optional | Pair-mode with Slice 2 (disjoint file surfaces). |
| 7 Release: tag + CHANGELOG + CI | none | Final commit + tag on trunk. |

## Commands reference

```bash
# create
git worktree add ../gowa-wt-<name> upgrade/v8.5.0-sync

# list
git worktree list

# remove (clean)
git worktree remove ../gowa-wt-<name>

# remove (with untracked files in worktree)
git worktree remove --force ../gowa-wt-<name>

# prune stale entries (after manual rm of a worktree dir)
git worktree prune
```

## Rollback discipline

Worktrees don't help with rollback granularity. Every slice's rollback is "branch reset to `pre-upgrade-snapshot-2026-05-14` + redeploy prior image" regardless of whether the slice was implemented in a worktree or on trunk. The Slice 4 worktree mainly serves staging isolation — for actual rollback, `git reset` happens on the trunk branch and the staging worktree is `worktree remove`'d.

## Why this differs from the original scaffold

The pre-D scaffold of this file prescribed "one worktree per slice (7 worktrees)" mirroring Dex's keynote framing literally. The consolidated P-stage plan made the actual dependency structure explicit: reset+reapply produces strict serial dependencies between slices, so per-slice worktrees mostly add ceremony. The W stage is most valuable at the two real isolation points — Slice 4 staging (mandatory) and optional Slice 2/6 parallelism — not as a uniform-per-slice ritual.
