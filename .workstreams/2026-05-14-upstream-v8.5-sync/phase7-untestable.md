# Phase 7 — untestable surfaces

Items in this list are fork-specific code paths that require either a paired
phone, a real `whatsmeow.Client` with a populated `Store`, or a live external
service to exercise faithfully. Marking them here keeps the scope honest:
Phase 7 covers everything mockable; the items below are gated by a future
human + paired-phone validation pass.

## `whatsmeow.Client` is a concrete struct, not an interface

The fork can't mock `whatsmeow.Client` without a refactor we're explicitly out
of scope for. Every code path that calls a method on `*whatsmeow.Client` is
partially testable only — the nil-client branch is covered, the real-client
branch is not.

Affected, with notes on what we DO cover:

| Function | Covered | Uncovered |
|---|---|---|
| `ValidateAndNormalizeJID` (pkg/utils) | nil-client passthrough, BR strip via `normalizePhoneBR`, group/newsletter/LID passthrough | `client.IsOnWhatsApp` round-trip — needs real client |
| `NormalizeJIDFromLIDWithContext` (infra/whatsapp) | non-LID passthrough, nil client safety check | `client.Store.LIDs.GetPNForLID` resolution — needs paired phone with LID mappings |
| `deduplicateLIDChats` (infra/whatsapp) | nil-client / nil-repo short-circuit | Real LID-resolution + MergeLIDChat orchestration — needs real client |
| `forwardHistorySyncCompleteToWebhook` (infra/whatsapp) | nil-client payload shape, whitelist gating | Device-ID derivation from `client.Store.ID` — needs paired client |

The dispatch and SQL primitives reachable from these functions are covered
exhaustively by separate tests:

- `MergeLIDChat`, `GetLIDChats` — `infrastructure/chatstorage/sqlite_repository_test.go`
- `forwardPayloadToConfiguredWebhooks` — `infrastructure/whatsapp/webhook_forward_test.go`
- HMAC signature — `pkg/utils/whatsapp_test.go` + e2e/`integration_test.go`

## Live WhatsApp `IsOnWhatsApp` validation

`ValidateAndNormalizeJID` with a non-nil client queries WhatsApp servers for
the canonical JID. There is no public mock layer for this RPC, and the
authoritative semantics (the BR 9th-digit-strip override based on WhatsApp's
own response) only manifest with a real connection.

Phase-7 substitute: assert the deterministic fallback (`normalizePhoneBR`)
matches the known correct shape for the 13-digit → 12-digit case. The real
validation gate is the human's paired-phone smoke test in Phase 8.

## `whatsmeow.Client.SetProxyAddress` wiring

`device_manager.go` reads `config.WhatsappProxyURL` and calls
`client.SetProxyAddress`. There is no fork-owned URL parser — `whatsmeow`
parses it. We can confirm:

- Empty `WhatsappProxyURL` skips the `SetProxyAddress` call.
- Non-empty value is forwarded to whatsmeow with the configured boolean flags.

…but only via a real `whatsmeow.Client` (the call site itself is a thin
forward). The config wiring through viper is already exercised by `cmd/root.go`
boot path. No fork-side logic to unit-test in isolation.

## Background goroutines (TTL sweepers)

`chatwoot.sentMessageIDs` and `groupNameCache` have background-goroutine
sweepers that run on package `init()`. They have indefinite lifetimes and the
TTL is 5 minutes — not pragmatically testable end-to-end. Covered via:

- Direct `MarkMessageAsSent` + manual `sync.Map` overwrite to simulate TTL
  expiry (chatwoot/client_test.go::TestIsMessageSentByUs_TTLExpiry).
- TTL semantics on `pkg/cache.Cache` via a short-TTL fresh instance
  (whatsapp/info_cache_test.go::TestInfoCache_UserInfo_TTLExpiry).

The sweeper goroutine itself is best-effort cleanup; missing a sweep tick is
not a correctness issue (the next check returns false anyway).

## What this means for the PR

The gaps above are gated by human validation, not by missing test code. Phase
7 ships:

- Every mockable boundary covered.
- One regression caught in slice-3 `GetLIDChats` (missing `archived` column —
  fixed in this phase).
- Documented gap list (this file) so Phase 8 has an explicit handoff.

When Phase 8 runs paired-phone validation, the items above are the checklist.
