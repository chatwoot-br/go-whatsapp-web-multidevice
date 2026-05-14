# Q — Questions

QRSPI Q stage. Decisions captured here gate the entire downstream pipeline. The user picks; no inferring.

---

## Q1 — Upgrade strategy

How to integrate 89 upstream commits (v8.1.2 → v8.5.0) into the fork?

- **A. Merge** `git merge upstream/main` into fork `main`. Preserves fork commit history; produces one large merge commit; conflicts resolved once.
- **B. Rebase** fork commits on top of `upstream/main`. Linear history; replays each fork delta as its own commit; conflict resolution per-fork-commit.
- **✓ C. Reset + re-apply.** Hard-reset `main` to `upstream/main`, then surgically re-apply fork deltas as fresh commits. Cleanest result; abandons fork commit history.
- **D. Cherry-pick selectively.** Skip whatsmeow churn / what isn't wanted; pick only specific upstream commits. Slowest; preserves maximum control.

**Answer:** C — Reset + re-apply deltas.

**Rationale:** Cleanest divergence story going forward. Fork commit history is preserved in the `pre-upgrade-snapshot-2026-05-14` tag for archaeology; new `main` is a small, auditable patch series on top of upstream `v8.5.0`. Each fork delta becomes a self-contained commit that can be PR'd upstream individually (relevant for Q3 BR phone rules, Helm chart, S3 fix).

---

## Q2 — Native chatwoot integration

Upstream now ships `src/infrastructure/chatwoot/{client,types}.go` + `src/ui/rest/chatwoot.go` + `webhook_forward.go` chatwoot-aware paths.

- **✓ A. Adopt + migrate.** Take upstream's chatwoot integration as primary; refactor `chatwoot-app` (the Rails side) to use it where it overlaps; retire bespoke webhook handlers that duplicate upstream behaviour.
- **B. Ignore.** Stay webhook-only; treat upstream's chatwoot code as dead code in the fork. Smallest blast radius; deferred re-evaluation.
- **C. Adopt selectively.** Take upstream's `chatwoot/client.go` + `types.go` (the boundary plumbing); skip the REST surface (`ui/rest/chatwoot.go`) if it duplicates what chatwoot-app already does.

**Answer:** A — Adopt + migrate chatwoot-app.

**Rationale:** Upstream now owns the integration contract; staying webhook-only means perpetually re-implementing what upstream maintains. Biggest blast radius this sprint, but the long-term divergence story collapses dramatically — chatwoot-app becomes a consumer of a stable upstream API instead of a custom webhook contract. **Implication: this work spans both repos** (gowa + chatwoot-app). The chatwoot-app cutover must be sequenced after Slice 4 lands in the gateway.

---

## Q3 — Phone normalization reconciliation

Upstream added `src/pkg/utils/phone.go` (generic). Fork has `ValidateAndNormalizeJID` (BR-specific, v8.1.0+7).

- **A. Drop fork's; submit BR rules upstream.** Use upstream's generic; open a PR on upstream adding BR rules. Long-term: zero divergence.
- **B. Keep fork's; ignore upstream's.** Don't touch what works; accept the duplication.
- **✓ C. Layer.** Keep upstream's generic as the base; the fork's BR rules become a thin override (`phone_br.go` calling into upstream's `phone.go`).

**Answer:** C — Layer fork's BR rules on top.

**Rationale:** Best balance. Upstream's `phone.go` becomes the base; fork ships `phone_br.go` (or equivalent) as a thin BR-specific override that wraps the upstream functions. Keeps the fork delta auditable as a single file. Sets up Option A as a future upstream PR with low effort — once `phone_br.go` is the only divergence, submitting it upstream becomes a single-file contribution.

---

## Q4 — Scope: what's *in* this upgrade vs deferred?

The 89 commits include features (healthcheck, CTWA Meta Ads, GIF playback, ghost mentions, archived chats filter, document thumbnails) plus ~20 whatsmeow protocol updates.

- **✓ A. All-in.** Take everything upstream ships in v8.5.0; defer nothing.
- **B. Protocol + integration only.** Whatsmeow updates (security-critical) + chatwoot integration + healthcheck. Defer feature commits (GIF, CTWA, ghost mentions) to a later sprint.
- **C. Minimal.** Only whatsmeow updates + healthcheck. Defer chatwoot integration adoption to a separate decision.

**Answer:** A — All-in.

**Rationale:** Q1=C makes this cheap. With reset+re-apply, "all-in" is just "main becomes upstream v8.5.0 + fork deltas"; there's no per-feature cherry-picking cost. Deferring features would *add* work, not save it. Slice 6 in the plan stays intact; consumers of new features (GIF playback, CTWA, ghost mentions) light up automatically.

---

## Open questions (not yet resolved)

- Will this upgrade ship as `v8.5.0+1` (fork-suffix convention) or as a new minor like `v8.5.0+0`?
- Does the deployed Chatwoot SaaS (chatwoot-app) need a coordinated cutover, or can the gateway upgrade ship independently?
- Are there in-flight customer integrations depending on the current webhook payload shape that would break under upstream's expanded event taxonomy?
