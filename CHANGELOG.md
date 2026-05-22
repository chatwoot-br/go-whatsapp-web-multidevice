# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v8.5.0+3] - 2026-05-22

### Fixed
- **rest: `GET /chat/:chat_jid/messages` returned HTTP 500 ("chat with JID ... not found") for every request.** Fiber's `UnescapePath` defaults to `false`, so the percent-encoded chat JID sent by the Chatwoot history-sync client (`...%40s.whatsapp.net`) reached the handler still encoded. The chatstorage lookup (`GetChatByDevice`) is an exact string match, so it never matched the stored JID (`...@s.whatsapp.net`); `chat == nil` raised an error that `PanicIfNeeded` converted to a recovered panic / HTTP 500. Every per-chat fetch during a Chatwoot history sync failed, so the sync completed having imported 0 messages. Enabled `UnescapePath` so path params are percent-decoded before routing — also fixes the same latent bug on `/chat/:chat_jid/{pin,disappearing,archive}`. Added regression test `TestRestFiberConfigDecodesEncodedChatJID` that drives the production fiber config.

## [v8.5.0+2] - 2026-05-21

### Fixed
- **chatstorage: `MergeLIDChat` deadlocked under `MaxOpenConns(1)`.** The transaction opened by `MergeLIDChat` held the only connection in the chatstorage pool (set by `cmd/root.go:initChatStorage`), then called `r.GetChatByDevice` twice — those helpers issue `r.db.QueryRow`, which requested a second connection and waited forever. After every history sync the fork-only `deduplicateLIDChats` goroutine triggers this path; if any `@lid` chat existed, the deadlock froze every subsequent `CreateMessage` from incoming WhatsApp events, and `handleWebhookForward` never fired (live messages stopped reaching downstream consumers ~5 s after the history-sync debounce). Inlined the two reads as `tx.QueryRow` so the whole transaction stays on the same connection; added a `MaxOpenConns(1)` invariant note in the function header and a regression test (`TestMergeLIDChat_NoDeadlockWithSingleConnPool`) that pins the behavior under the production pool size.

## [v8.5.0+1] - 2026-05-14 (Synced with upstream v8.5.0)

### Upstream Changes
- ~31 whatsmeow protocol updates (security/compatibility patches)
- Native Chatwoot integration: src/infrastructure/chatwoot/{client,sync,sync_test,sync_types,types}.go, src/ui/rest/chatwoot.go, /chatwoot/webhook + /chatwoot/sync* endpoints, 8 CHATWOOT_* env vars
- LID handling improvements: ResolveLIDToPhone/ResolvePhoneToLID primitives, LID-aware auto-reply, group-participants phone fix
- Webhook taxonomy: chat_presence (typing), call.offer (incoming call), contacts_array shape, media captions in payloads, is_from_me top-level field
- feat: healthcheck endpoint, GIF playback, document thumbnails, CTWA Meta Ads referral support, ghost mentions, archived chats filtering
- fix: Docker permission readonly DB on group messages, document thumbnail security, audio extension test parser
- chore: Go 1.25 / Alpine 3.23, dependency updates

Full upstream commit log: git log v8.1.2..v8.5.0

### Fork Changes
- slice 1: reset to upstream/v8.5.0 + reapply release rail (Helm chart, 4 CI workflows, Dockerfile mailcap)
- slice 2: BR phone normalization layer (src/pkg/utils/phone_br.go) + 39 caller sweep across src/usecase/{send,group,message,chat,user,newsletter}.go (preserves v8.1.0+7 ValidateAndNormalizeJID behavior on upstream baseline)
- slice 3: LID dedup + history_sync_complete dispatch (preserves v8.1.0+6 deduplicateLIDChats post-history-sync pass + MergeLIDChat/GetLIDChats chatstorage primitives + NormalizeJIDFromLIDWithContext 30s-timeout variant; new file forward_history_sync.go scopes the fork-specific event)
- slice 6: fork-only delta sweep (proxy support v8.1.0+3, info-request cache v8.1.0+2, S3 image-extension fix v8.1.0+5; OQ8 device_manager.go own-commit due to upstream/any-modernization overlap; audio/PTT v8.1.0+1 and APP_BASE_PATH v8.1.0+1 subsumed by upstream)
- slice 4: chatwoot lockstep cutover gateway-side wiring (8 CHATWOOT_* env vars in Helm + configmap; /chatwoot/webhook route order verified; chatwoot-app Rails cutover documented as separate-repo follow-up)
- slice 5: webhook taxonomy + env audit (docs/webhook-payload.md union of upstream events + fork's history_sync_complete; WHATSAPP_WEBHOOK_INCLUDE_OUTGOING marked deprecated; is_from_me echo-suppression documented)
- fix(webhook): recovered chat_name + sender_name payload fields (v8.1.0+1) missed during slice 6 sweep

### Known follow-ups (out of scope for this upgrade)
- chatwoot-app Rails-side cutover PR (Channel::Whatsapp::Provider rewire)
- Paired-phone validation: trigger each event from staging phone; confirm receipt at test webhook
- Cleanup: usecase callers of new info_cache helpers (left dormant but build-green per Slice 6 agent note)

## [v8.1.2+1] - 2026-01-26 (Synced with upstream v8.1.2)

### Upstream Changes
- feat: add webhook events for newsletters and group.joined
- fix: react to other users' messages by looking up IsFromMe from database (#535)
- fix: webhook event whitelist filtering for groups and proper event names (#539)
- fix(security): prevent cross-device data leak in chat message queries (#525)
- fix(device): sort device list by created_at for stable UI ordering (#528)
- fix: store phone-sent messages in chat history (issue #526) (#530)
- chore: update dependencies (golang.org/x/text to v0.33.0, app version to v8.1.2)

### Fork Changes
- chore: update whatsmeow to latest (v0.0.0-20260126173513-4dbbef8d4d4a)
- fix(docker): add mailcap package for MIME types database

## [v8.1.0+7] - 2026-01-20

### Fixed
- fix(utils): normalize Brazilian phone numbers to prevent duplicate contacts
  - Add ValidateAndNormalizeJID function that handles Brazilian 9-digit mobile normalization
  - Update all callers across send, chat, group, message, newsletter, and user usecases
  - Mark ValidateJidWithLogin as deprecated in favor of the new function

## [v8.1.0+6] - 2026-01-18

### Fixed
- fix(history-sync): resolve LID duplicate chats and context cancellation
  - Add dedicated context with 5s timeout for LID resolution to prevent context cancellation errors
  - Add NormalizeJIDFromLIDWithContext helper for isolated LID lookups
  - Add MergeLIDChat to chatstorage for deduplicating chats with same sender but different JID formats
  - Add post-sync deduplication to merge LID chats after history sync

## [v8.1.0+5] - 2026-01-17

### Fixed
- fix(utils): derive image extension from Content-Type for S3 URLs

## [v8.1.0+4] - 2026-01-15

### Added
- feat(whatsapp): enable full history sync and ON_DEMAND capability
- feat(whatsapp): handle unavailable messages from linked devices
- feat(whatsapp): process ON_DEMAND history sync responses
- feat: add logs directory to .gitignore and create .keep file

### Fixed
- fix(whatsapp): normalize chat_id from LID to phone number in webhook

### Changed
- chore: update dependencies for go.mau.fi/whatsmeow and golang.org/x packages

## [v8.1.0+3] - 2026-01-13

### Added
- feat(proxy): add SOCKS5/HTTP/HTTPS proxy support for WhatsApp connections
- feat(proxy): display external proxy IP in device card UI

### Fixed
- fix(webhook): include caption in payload when auto-downloading media (image, video, video_note)

## [v8.1.0+2] - 2026-01-08

### Added
- feat(cache): add short-term caching for info requests
- feat(webhook): update events list to include history_sync_complete and improve documentation

### Fixed
- fix(cache): cache error responses to prevent repeated API calls
- fix(send): use LID for message delivery with targeted approach
- Various CI workflow fixes for tag patterns and multi-arch builds

### Changed
- refactor(workflow): trigger Helm chart release on version tags only

## [v8.1.0+1] - 2025-01-07

### Added
- feat(helm): add gowa Helm chart for Kubernetes deployment
- feat(webhook): add chat_name to outgoing message payload
- feat(chat): add sender_name field for group message contacts
- feat(whatsapp): add history sync webhook notification
- feat(audio): add OGG Opus conversion for PTT voice notes
- feat(whatsapp): include is_from_me in webhook payload
- feat: add multi-device support guide documentation
- feat: add waveform generation for audio messages
- feat: enhance audio handling with MIME type resolution and duration retrieval

### Fixed
- fix(whatsapp): debounce history sync webhook to wait for all events
- fix(login): use background context for QR channel
- fix(device.go): Fix DeviceMiddleware to allow if APP_BASE_PATH is changed

### Changed
- Updated GitHub Actions workflows to support fork versioning (v8.1.0+1 format)
- Added chart-releaser workflow for Helm chart releases

