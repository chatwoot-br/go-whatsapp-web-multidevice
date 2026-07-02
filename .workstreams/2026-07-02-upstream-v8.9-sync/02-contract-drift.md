# Contract-drift check — upstream v8.9.0 merge (2026-07-02)

Verdict: **0 breaking / 0 behavioral / HMAC stable**. Clean.

## Method

Diffed merge-base `131b99b` → `upstream/v8.9.0` over every webhook-producing surface, compared
against what `chatwoot-app`'s fork controller parses
(`fork/app/controllers/webhooks/whatsapp_web_controller.rb` + incoming-message service).

## Findings

1. **Webhook forwarding files: empty diff.** `git diff 131b99b upstream/v8.9.0 -- 'src/infrastructure/whatsapp/webhook*' 'event_*' 'call*' 'forward*'` produced no output — upstream did not touch `forwardToWebhooks`, event payload construction, or call-event forwarding in this range. Top-level `device_id`/`event`/`payload` shape, the `event` switch values, and the fork's `chat_name`/`sender_name` fields are untouched by construction.
2. **HMAC signing untouched.** Upstream's +101 lines in `src/pkg/utils/whatsapp.go` are `ExtractMediaInfo` gaining a `directPath` return + new `ResolveMediaDirectPath`/`BuildDownloadableMessage` helpers (#731 media persistence). No hmac/sha256/signature/digest code touched; `X-Hub-Signature-256` unchanged. Fork's known-vector test (`TestGetMessageDigestOrSignature_KnownVector`) passes post-merge (gate 3).
3. **Call-reject API (#735) is inbound-only.** `POST /call/reject` (new `ui/rest/call.go`, `usecase/call.go`, `domains/call/`) consumes values from the existing `call.offer` webhook; it emits nothing. The `docs/webhook-payload.md` addition documents the pre-existing `call.offer` payload — not a shape change.
4. **`history_sync_complete` (fork-only) intact** — `forward_history_sync.go` untouched by the merge; emit verified post-merge.
5. **`direct_path` is storage-internal.** New column in `messages` + threading through `sqlite_repository.go`; it is not added to any webhook payload map.

No action needed in `chatwoot-app`.
