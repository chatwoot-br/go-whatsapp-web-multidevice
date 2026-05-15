# Q2 reversal — keep fork's custom chatwoot integration

Date: 2026-05-15. Decision owner: human (explicit pick).

## What changed

D-review locked **Q2 = A (Adopt + migrate chatwoot-app)** on the rationale:
*"upstream now owns the integration contract; staying webhook-only means
perpetually re-implementing what upstream maintains."*

The topology investigation (`09-chatwoot-app-topology.md`) falsified both
halves of that rationale:

1. **Not equivalent — poorer.** GoWA-native chatwoot is a 5-feature
   regression vs the fork's current custom integration: LID phone-discovery
   (3 scenarios), reaction-on-original-message, status-progression
   pessimistic locking, group-sender contacts, two-phase history-sync
   normalization. Adopting it downgrades a live WhatsApp support system.
2. **Low maintenance cost, not high.** The fork's WhatsApp channel is
   entirely fork-owned under `fork/**`, **zero rebase-surface tax**
   (`cw:fork/bin/check-rebase-surface.sh:70-83` allowlist has no WhatsApp
   file; injected via prepend overlay + route-append). The "perpetual
   re-implementation" Q2 feared doesn't materialize — the integration is
   cleanly isolated and doesn't fight upstream rebases.

## New decision

**Q2 → reversed. Keep the fork's custom bidirectional chatwoot integration.
Do NOT activate GoWA-native chatwoot.**

## Consequences

- **Gateway upgrade (33 commits): unaffected.** All fork deltas preserved;
  the fork's *standard* custom webhook output (`chat_name`, `sender_name`,
  `history_sync_complete`, etc.) is the active path. Sound as-is.
- **Slice 4 (CHATWOOT_* env wiring + Helm): kept dormant, not reverted.**
  The native chatwoot module arrived via the upstream v8.5.0 reset — it is
  upstream code. Ripping it out re-introduces rebase divergence (the exact
  tax this whole upgrade avoids). `CHATWOOT_ENABLED` defaults false → the
  module is inert. Documented as "present but dormant; fork's custom
  integration is the active path." Minimal-divergence choice, consistent
  with the reset+reapply philosophy.
- **chatwoot-app side: NO topology rewrite.** No port of 5 features, no
  data migration, no native-API cutover. The Model-3 native path is not
  taken.
- **Remaining work shrinks to a contract-drift check.** The real
  "chatwoot integration ready" gate under this decision: does the
  post-v8.5.0 GoWA *standard* (non-native) webhook output still match what
  chatwoot-app's fork webhook controller parses? The 89-commit upstream
  sync may have renamed/changed adjacent base-payload fields, expanded the
  event taxonomy, or shifted `is_from_me`/outgoing-forwarding semantics
  (INCLUDE_OUTGOING deprecated → outgoing always forwarded). That delta
  must be verified; any break is a small fork-owned chatwoot-app fix
  (zero rebase risk).

## Supersedes

- `00-questions.md` Q2 answer (A) — now reversed; original kept for trail.
- `04-plan.md` Slice 4 framing — gateway wiring stays but is dormant, not
  a lockstep cutover. Slice 4's "chatwoot-app cutover" sub-task is dropped.
- `08-pr-description.md` "Known follow-up #1 (chatwoot-app Rails cutover)"
  — replaced by the contract-drift check below.
