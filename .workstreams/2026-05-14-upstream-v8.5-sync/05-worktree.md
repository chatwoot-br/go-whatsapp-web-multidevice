# W — Worktree

QRSPI W stage. Isolated `git worktree add` per implementation slice. Parallelisable; clean rollback.

## Worktree per slice

| Slice | Worktree path | Branch | Status |
|---|---|---|---|
| Slice 0 (preflight) | _(none — main)_ | `upgrade/v8.5.0-sync` | pending |
| Slice 1 (whatsmeow) | `../gowa-wt-slice1-whatsmeow` | `upgrade/v8.5.0-sync-s1` | pending |
| Slice 2 (healthcheck) | `../gowa-wt-slice2-healthcheck` | `upgrade/v8.5.0-sync-s2` | pending |
| Slice 3 (LID + phone) | `../gowa-wt-slice3-lid-phone` | `upgrade/v8.5.0-sync-s3` | pending |
| Slice 4 (chatwoot) | `../gowa-wt-slice4-chatwoot` | `upgrade/v8.5.0-sync-s4` | pending |
| Slice 5 (webhooks) | `../gowa-wt-slice5-webhooks` | `upgrade/v8.5.0-sync-s5` | pending |
| Slice 6 (features) | `../gowa-wt-slice6-features` | `upgrade/v8.5.0-sync-s6` | pending |

## Commands

```bash
# create worktree for a slice (run from main repo)
git worktree add ../gowa-wt-slice1-whatsmeow -b upgrade/v8.5.0-sync-s1 main

# work happens in the worktree; no other tabs touch the main checkout
cd ../gowa-wt-slice1-whatsmeow

# when slice ships:
git worktree remove ../gowa-wt-slice1-whatsmeow
```

## Why worktrees, not branches

- Multiple slices can be implemented in parallel without `git stash` dance.
- Each slice has its own running gateway process (different ports if needed) for end-to-end testing.
- A blown slice is `worktree remove`, not a hard-reset on the shared checkout.
