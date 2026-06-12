# PR: Sync fork to upstream v8.7.0 + main tail (release v8.7.0+1)

**Branch:** `upgrade/v8.7.0-sync` → `main` · **Release:** `v8.7.0+1` (local tag) · **Method:** merge (no rebase)

## Summary

Brings `chatwoot-br/go-whatsapp-web-multidevice` from upstream **v8.5.0** (`v8.5.0+5`) up to the
latest upstream release **v8.7.0** **plus** upstream's post-`v8.7.0` `main` tail (the still-unreleased
work upstream version-bumped to an in-dev `v8.8.0`). The fork release rail follows the upstream
**release tag**, so this ships as **`v8.7.0+1`**. Two merges: Phase A = `upstream/v8.7.0`
(chatwoot-free core), Phase B = `upstream/main` tail (chatwoot + webhook `session_id`). All fork
features preserved; GoWA-native Chatwoot stays dormant (`CHATWOOT_ENABLED=false`).
**0 breaking changes to the chatwoot-app contract.**

## What's included from upstream

- **whatsmeow** `v0.0.0-20260513` → `v0.0.0-20260609`; **Go 1.25.5**; new **pure-Go SQLite**
  (`modernc.org/sqlite`, build-tag selected via `pkg/sqlite`); fiber/fasthttp/libsignal bumps.
- Message **reactions** (persist + `message.reaction` webhook + history-sync reactions),
  `SecretEncryptedMessage{MESSAGE_EDIT}` decryption, label appstate webhooks, presence pulse,
  463 send-retry, quoted media replies, ARMv7 build path, **`session_id`** webhook field,
  chat-list name fallback (#675), saved-contact-name fix (#688).

## Notable reconciliation decisions (full analysis: `02-contract-drift.md`)

1. **Chatwoot custom attribute `waha_whatsapp_jid` → `gowa_whatsapp_jid`.** Adopt upstream's
   `gowa` for **writes**, but **read both** (gowa, then `waha` fallback) in
   `client.go:FindContactByIdentifier` and the inbound route `ui/rest/chatwoot.go`. Existing
   fork contacts (stored under `waha`, matched by Identifier/phone, never rewritten) keep routing
   agent replies — **no data migration**. Cross-repo grep (chatwoot-app/operator/gitops) was clean.
2. **`FindOrCreateContact` preserves existing 1:1 names** (fills blanks only; groups still refresh)
   — adopts upstream #675/#688 over the fork's always-overwrite. Behavioral change, owned & documented.
3. **`session_id`** is an additive top-level webhook key (chatwoot-app ignores unknown keys); HMAC
   `X-Hub-Signature-256` and fork fields `chat_name`/`sender_name`/`history_sync_complete` intact.

## Fork features preserved

BR phone normalization, LID dedup + `history_sync_complete`, full history sync + `ON_DEMAND`,
info cache, SOCKS/HTTP/HTTPS proxy, webhook taxonomy + HMAC, `InitWaDB` bounded retry.

## Conflicts resolved

20 across both merges (incl. `history_sync.go` reaction-first reorder, `event_message.go`
chat_name + secret-edit, `database.go` retry + pure-Go sqlite, go.mod, CI workflows, 4 add/add
test files unioned). Dropped-function `NormalizeJIDFromLID` rewired to the fork's
`NormalizeJIDFromLIDWithContext`. `git rerere` recorded all resolutions.

## Validation (this environment — all green)

`go build ./... && go vet ./... && go test ./... && go test -tags=e2e ./...` — **all pass**
(logs: `.workstreams/2026-06-11-upstream-v8.7-sync/logs/`). Coverage: chatwoot 61%, pgimport 78%,
chatstorage 43%, utils 47%, whatsapp 34%.

## Human-owned gates (before merge to `main`)

- [ ] **Paired-phone + live Chatwoot UAT** — confirm inbound agent reply routes for **both** a
      new contact (`gowa_whatsapp_jid`) and a pre-existing one (`waha` fallback); confirm
      `session_id` benign; confirm name-preserve matches expectations. (`untestable-surfaces.md`)
- [ ] **Push** branch + `v8.7.0+1` tag (a pushed `v*` tag triggers `release.yml` → image).
- [ ] Optional: add `waha` fallback to `pgimport` upsert SQL **iff** direct-DB import runs on legacy data.
