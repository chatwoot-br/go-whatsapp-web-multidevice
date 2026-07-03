# CHAT STORAGE

Generated: 2026-06-06

## OVERVIEW

`SQLiteRepository` implements chat, message, edit-history, call, reaction, statistic, schema, and device-record storage behind `domains/chatstorage.IChatStorageRepository`.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Repository contract | `../../domains/chatstorage/interfaces.go` | Any method addition must be implemented here and in WhatsApp wrapper. |
| SQL implementation | `sqlite_repository.go` | Single large repository file. |
| Migrations | `sqlite_repository.go` `getMigrations()` | Append-only list, currently 30 migrations (v8.9.0 added `direct_path`). |
| Message edit history | `sqlite_repository.go`, `sqlite_repository_edit_test.go` | `message_edits` is append-only history while original message content updates. |
| Chatwoot links | `sqlite_repository.go`, `../../domains/chatstorage/chatstorage.go` | Maps WhatsApp and Chatwoot IDs for idempotency, read/delete sync, and webhook routing. |
| Chatwoot retry queue | `sqlite_repository.go` | Persists live forward retry jobs across restarts. |
| Tests | `sqlite_repository_test.go`, `sqlite_repository_edit_test.go` | Add coverage for schema/data isolation changes. |

## CONVENTIONS

- Default chat storage URI is `file:storages/chatstorage.db`; connection setup is in `cmd/root.go`.
- `chats` primary key is `(jid, device_id)`; `messages` primary key is `(id, chat_jid, device_id)`.
- `GetMessages` and `SearchMessages` fail fast if device ID is missing.
- Use `GetMessageByIDAndDevice` for device-scoped ID lookups such as quoted replies.
- Use `GetChatByDevice`, `DeleteChatByDevice`, `DeleteMessageByDevice`, and count-by-device variants for scoped flows.
- `chatwoot_message_links` primary key is `(device_id, wa_message_id)`; link lookups by Chatwoot ID and unread chat are indexed.
- `chatwoot_forward_queue` uniqueness is `(device_id, event_name, wa_message_id)`; cleanup paths must include it.
- `CreateMessage` and sent-message storage derive the current device identity from the whatsmeow client context.
- **Runs at `SetMaxOpenConns(1)` (see `cmd/root.go`, default `config.ChatStorageMaxOpenConns=1`).** `StoreChat`/`StoreMessage`/`StoreReaction` emulate upserts as UPDATE-then-INSERT with no transaction and no `ON CONFLICT`, and `MergeLIDChat` does tx-scoped reads (it must — a second pooled connection inside its tx would self-deadlock). Both are only correct with a single connection: >1 lets two writers race the same primary key and silently drop the loser's row. Upstream keeps raising this default (v8.9.0/#732 → 5); keep it 1 until the upserts are made atomic (`INSERT … ON CONFLICT DO UPDATE`, as `EnqueueChatwootForwardEvent` already does).
- `status@broadcast` must always produce display name `Status`.
- Storage tests use real SQLite drivers, including temp DB and in-memory variants.

## ANTI-PATTERNS

- Do not add new user-facing query paths that use `GetChat`, `DeleteChat`, or `GetMessageByID` without confirming device isolation.
- Do not raise `ChatStorageMaxOpenConns` above 1 while the upserts remain non-atomic — it is a data-loss race, not a perf knob (see CONVENTIONS).
- Do not reorder or edit old migrations for a live DB; append a new migration.
- Do not build SQL with untrusted string interpolation. Existing dynamic clauses use fixed fragments plus args.
- Do not forget the device registry operations when changing purge/load behavior.
- Do not leave Chatwoot link or retry rows behind when truncating all chats or deleting one device.
