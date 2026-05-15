> **SUPERSEDED 2026-05-15** — Q2 reversed (`.workstreams/2026-05-14-upstream-v8.5-sync/10-q2-reversal.md`). The fork keeps its custom chatwoot integration; GoWA-native is NOT adopted. The Rails-side cutover described below is NOT required. Webhook contract verified clean — see `11-contract-drift.md`. Retained for trail.

---

# chatwoot-app lockstep cutover (Slice 4 follow-up)

**Date**: 2026-05-14
**Status**: Decided (Q2 = A, lockstep). Gateway side wired in Slice 4 of `upgrade/v8.5.0-sync`. Rails-side change OUT OF SCOPE for this repo.

## Decision

Cut over the Rails `chatwoot-app` consumer in lockstep with the gateway's v8.5.0-sync release. No transitional dual-shape parsing on either side.

After the gateway upgrade lands:

- The gateway forwards Chatwoot push payloads to `POST /webhooks/whatsapp/:hmac_token` using upstream's v8.5.0 shape — message events always include outgoing traffic; the `is_from_me` field on the payload disambiguates direction.
- The Rails-side `Channel::Whatsapp::Provider#process_messages` (in the `chatwoot-app` repo) must switch from its legacy custom-shape parser to consume upstream's shape.
- Echo suppression (don't re-broadcast our own sends back into the inbox) MUST be implemented on the consumer side by checking `is_from_me` on the payload, NOT by toggling `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` on the gateway. That env knob is **deprecated in v8.5**: outgoing webhooks are always forwarded; the consumer is responsible for filtering.

## What this PR (this repo) does

- Wires the 8 `CHATWOOT_*` env vars end-to-end (`src/.env.example`, `src/config/settings.go`, `src/cmd/root.go`, Helm `values.yaml`, Helm `configmap.yaml` + `secret.yaml`).
- Confirms upstream's chatwoot module (`src/infrastructure/chatwoot/`) and REST handler (`src/ui/rest/chatwoot.go`) are present.
- Confirms `3b87f4e` route ordering: `/chatwoot/webhook` is registered BEFORE the basic-auth middleware in `src/cmd/rest.go`, so Chatwoot's HMAC-authenticated webhook bypasses gateway basic auth.
- Marks `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` deprecated in `src/.env.example`.

## What this PR does NOT do (separate chatwoot-app PR required)

- Update `Channel::Whatsapp::Provider#process_messages` to parse upstream v8.5 payloads.
- Add `is_from_me` echo-suppression on the consumer side.
- Migrate any production deployment env files away from `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING`.
- Paired-phone staging validation of the end-to-end flow.

## Operational sequencing

1. Ship this gateway PR; do not deploy to prod yet.
2. Ship the corresponding `chatwoot-app` PR (Rails consumer rewrite).
3. Stage-deploy both repos with a paired test phone; verify inbound + outbound + echo-suppression.
4. Coordinated prod deploy (gateway and chatwoot-app together). Rollback plan: revert both repos in lockstep.

## References

- Gateway commit `3b87f4e` — chatwoot webhook + basic-auth route order fix (PR #565).
- `src/.env.example` — `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` carries the deprecation comment.
- `.workstreams/2026-05-14-upstream-v8.5-sync/07-execution.md` — Phase 4 entry.
