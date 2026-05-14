# R — Research (ticket hidden)

QRSPI R stage. **Open this file in a fresh chat with the goal "characterize the divergence between fork v8.1.2+1 and upstream v8.5.0; produce facts only."** Do not paste the upgrade ticket or design intent — keep this stage opinion-free.

Output goes here, structured as:

## Source facts

- Upstream branch HEAD: `17af98e` (`upstream/main`, `git describe` → `v8.5.0-5-g17af98e`; tag `v8.5.0` = `2e1798b`)
- Fork branch HEAD: `b4fd010` (`main`, tagged `v8.1.2+1`; base tag `v8.1.2` = `097403d`)
- Merge-base `main` ↔ `upstream/main`: `48d9be8` (one commit before `v8.1.2`).
- Commit count delta: `main..upstream/main` = **88**; `upstream/main..main` = **41**; `v8.1.2..main` = **42**; `v8.1.2..upstream/main` = **89**.
- Files changed `main..upstream/main`: **134 files, +6841 / −8830 lines** (`git diff --shortstat`).
- Files changed `v8.1.2..upstream/main`: **96 files, +7064 / −2204 lines**.
- Files changed `v8.1.2..main`: **77 files, +7096 / −247 lines**.

## Commit taxonomy

Bucketing applied to the 88 commits in `main..upstream/main`. Several commits land in multiple buckets — overlaps are listed explicitly. Two commits (`381c381`, `a8b5ed8`) have no conventional prefix and are uncategorised by the prefix-based buckets but match no special-topic bucket either.

- **whatsmeow protocol updates: 31** (subject contains `whatsmeow`).
  - 26 direct `chore: update whatsmeow [...]`: `17af98e`, `495beb0`, `733d245`, `11b136e`, `f30fdf3`, `e61bc61`, `1e2b160`, `e774dde`, `a7ed824`, `ae4bd12`, `2ce3d8f`, `c9faa4d`, `0e0af10`, `8cb167d`, `bb1b3ff`, `b9f8b1d`, `33b0bab`, `3368c45`, `e1b9531`, `fc4c7f4`, `c923bc9`, `5de0d2d`, `b521351`, `162dde0`, `6e6bee5`, `9710533`.
  - 5 `chore: update Go dependencies (whatsmeow + …)` / similar: `875bf5e`, `39cd285`, `72a2e98`, `bed50e3`, `77d9652`. (`d3554d7` also mentions whatsmeow in a docs commit body — see docs/chores.)
- **chatwoot integration: 8** (subject contains `chatwoot` OR commit touches `src/infrastructure/chatwoot/`).
  - Subject match (5): `2d27ea8`, `3b87f4e`, `29907ee`, `909b6e6`, `44a128c`.
  - Path-touch only (3): `2e1798b`, `33b0bab`, `33b6509`. New chatwoot files added under `src/infrastructure/chatwoot/`: `client.go`, `sync.go`, `sync_test.go`, `sync_types.go`, `types.go` (`git diff --name-status main..upstream/main`).
- **webhook contract changes: 6** (subject contains `webhook`): `437df12`, `c428afa`, `00ee65b`, `306391e`, `3b87f4e`, `29907ee`. (`3b87f4e` and `29907ee` also in chatwoot bucket.)
- **LID handling: 3** (subject contains `LID` / `lid` / `@lid`): `40b0875`, `d718ef8`, `17ff32f`. (`40b0875` and `d718ef8` also in fix bucket; `17ff32f` also in feat bucket.)
- **new features (`feat:` / `feat(`): 19**: `437df12`, `4f04285`, `536f5fb`, `fe7d2c7`, `a6b6a02`, `fc7fe7b`, `a05b696`, `ea27ef2`, `c428afa`, `9dddc5d`, `e13966a`, `00ee65b`, `61c29b0`, `17ff32f`, `c7a182c`, `3b87f4e`, `5c193bc`, `909b6e6`, `44a128c`.
- **bug fixes (`fix:` / `fix(`): 17**: `66d25e8`, `a6f7b44`, `40b0875`, `2d27ea8`, `75869b5`, `432e974`, `d718ef8`, `8606995`, `3af045c`, `33b6509`, `8eb40a5`, `d14e997`, `3166540`, `71cb1b9`, `1c50d63`, `5b4cc5c`, `306391e`.
- **docs / CI / chores (`docs:` / `chore:` / `ci:` / `refactor:` / `build:`): 50**: all 31 whatsmeow commits plus `2e1798b`, `d32aadf`, `b7bbe67`, `0b8dead`, `fa02384`, `aef96a7`, `a05a443`, `06b51ad`, `a516049`, `cd9b507`, `b1cbfc1`, `63c316d`, `4c500a4`, `f2cf681`, `b42727e`, `29907ee`, `c1831f7`, `dfccc04`, `d3554d7`.

Bucket-overlap notes (commit → buckets):

- `3b87f4e` → chatwoot, webhook, feat.
- `29907ee` → chatwoot, webhook, docs.
- `2d27ea8` → chatwoot, fix.
- `33b0bab` → chatwoot (path-touch), whatsmeow, chores.
- `33b6509` → chatwoot (path-touch), fix.
- `2e1798b` → chatwoot (path-touch), chores.
- `437df12`, `c428afa`, `00ee65b` → webhook, feat.
- `306391e` → webhook, fix.
- `40b0875`, `d718ef8` → LID, fix.
- `17ff32f` → LID, feat.
- All 31 whatsmeow commits also count under docs/chores via their `chore:`/`refactor:`/`build:`/`docs:` prefixes.
- `381c381` ("Persist incoming contact messages in chatstorage") and `a8b5ed8` ("Persist incoming calls to chat storage") have no conventional prefix — they appear only as raw commits, in none of the 7 buckets.

## File-level overlap inventory

Intersection of files touched in `v8.1.2..main` (77 files) and `v8.1.2..upstream/main` (96 files) = **37 files**. "Touched more on" = greater commit count on that side (`git log --oneline <range> -- <file>` per side); `tie` = equal counts.

| Overlap file | Fork commits | Upstream commits | Touched more on |
|---|---:|---:|---|
| `.github/workflows/build-docker-image.yaml` | 5 | 1 | fork |
| `.github/workflows/release.yml` | 4 | 2 | fork |
| `.github/workflows/set-latest-tag.yaml` | 1 | 1 | tie |
| `.gitignore` | 3 | 1 | fork |
| `docker/golang.Dockerfile` | 1 | 3 | upstream |
| `docs/webhook-payload.md` | 4 | 9 | upstream |
| `readme.md` | 2 | 8 | upstream |
| `src/.env.example` | 3 | 6 | upstream |
| `src/cmd/root.go` | 1 | 7 | upstream |
| `src/config/settings.go` | 9 | 14 | upstream |
| `src/domains/chat/chat.go` | 1 | 2 | upstream |
| `src/domains/chatstorage/interfaces.go` | 2 | 3 | upstream |
| `src/go.mod` | 4 | 35 | upstream |
| `src/go.sum` | 1 | 34 | upstream |
| `src/infrastructure/chatstorage/device_repository.go` | 2 | 4 | upstream |
| `src/infrastructure/chatstorage/sqlite_repository.go` | 2 | 5 | upstream |
| `src/infrastructure/whatsapp/auto_reply.go` | 1 | 3 | upstream |
| `src/infrastructure/whatsapp/chatstorage_wrapper.go` | 2 | 3 | upstream |
| `src/infrastructure/whatsapp/device_manager.go` | 3 | 2 | fork |
| `src/infrastructure/whatsapp/event_group.go` | 1 | 1 | tie |
| `src/infrastructure/whatsapp/event_handler.go` | 3 | 6 | upstream |
| `src/infrastructure/whatsapp/event_message_handler.go` | 2 | 3 | upstream |
| `src/infrastructure/whatsapp/event_message.go` | 5 | 7 | upstream |
| `src/infrastructure/whatsapp/event_newsletter.go` | 1 | 1 | tie |
| `src/infrastructure/whatsapp/history_sync.go` | 5 | 2 | fork |
| `src/infrastructure/whatsapp/init.go` | 1 | 1 | tie |
| `src/pkg/utils/general_test.go` | 1 | 2 | upstream |
| `src/pkg/utils/general.go` | 1 | 3 | upstream |
| `src/pkg/utils/whatsapp_test.go` | 1 | 3 | upstream |
| `src/pkg/utils/whatsapp.go` | 2 | 11 | upstream |
| `src/ui/rest/chat.go` | 1 | 1 | tie |
| `src/usecase/app.go` | 1 | 1 | tie |
| `src/usecase/chat.go` | 3 | 2 | fork |
| `src/usecase/group.go` | 3 | 2 | fork |
| `src/usecase/message.go` | 3 | 3 | tie |
| `src/usecase/send.go` | 4 | 8 | upstream |
| `src/usecase/user.go` | 3 | 4 | upstream |

Side tallies: upstream-heavier on **23/37**, fork-heavier on **8/37**, tied on **6/37**.

## Webhook contract diff (factual only)

Scope: `docs/webhook-payload.md`, `readme.md`, `src/.env.example`, and `src/infrastructure/whatsapp/webhook*.go` between `v8.1.2..upstream/main` and `v8.1.2..main`; struct/field-level inspection of webhook source.

### Event types

- "Available Webhook Events" table in `docs/webhook-payload.md` (`git show <ref>:docs/webhook-payload.md`):
  - In upstream (B) but not fork (A): `chat_presence`, `call.offer`. Added by `c428afa` (`feat: forward incoming ChatPresence (typing) events via webhook (#547)`) and `5c193bc` (`feat: add auto-reject incoming calls feature (#563)`).
  - In fork (A) but not upstream (B): `history_sync_complete` (introduced on fork side; appears only in `main` doc table, not in `upstream/main`).
- Doc-section additions in upstream `docs/webhook-payload.md` `v8.1.2..upstream/main` (`git diff v8.1.2..upstream/main -- docs/webhook-payload.md`): new top-level sections `## Chat Presence Events` and `## Call Events`, and new sub-sections under existing groups (`### Contacts Array Message`, `### Meta Ads Referral (Click-to-WhatsApp)`). Total shortstat: 1 file, +473 / −4.

### Top-level payload keys

- Top-level keys identical in both: `event`, `device_id`, `payload` (per `### Top-Level Fields` table, both refs).
- In B but not A:
  - none at the top level.
  - Under `### Common Payload Fields`: `is_from_me` (boolean) — added in upstream only (diff of that subsection, `v8.1.2..upstream/main`).
- In A but not B:
  - none at the top level (`docs/webhook-payload.md`).

### Env var names (`WEBHOOK_*` / `CHATWOOT_*`)

`src/.env.example` `v8.1.2..upstream/main` adds (`+`-only lines): `WHATSAPP_AUTO_REJECT_CALL`, `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING`, `WHATSAPP_PRESENCE_ON_CONNECT`, `WHATSAPP_CHAT_STORAGE`, `CHATWOOT_ENABLED`, `CHATWOOT_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`, `CHATWOOT_INBOX_ID`, `CHATWOOT_DEVICE_ID`, `CHATWOOT_IMPORT_MESSAGES`, `CHATWOOT_DAYS_LIMIT_IMPORT_MESSAGES`. `readme.md` adds the same eight `CHATWOOT_*` env-table rows plus `CHATWOOT_IMPORT_CONTACTS` (mentioned in commit body of `3b87f4e`, exposed via the README diff). Introducing commits: `44a128c` (initial chatwoot integration), `3b87f4e` (sync flags + webhook auth fix), `b42727e` (auto-reject README), `61c29b0` (presence-on-connect), `c428afa` (chat-presence).
- B but not A (`grep -iE 'CHATWOOT_|WHATSAPP_AUTO_REJECT|WEBHOOK_INCLUDE_OUTGOING|PRESENCE_ON_CONNECT' main:src/.env.example` → empty; fork's `main:src/.env.example` contains only `WHATSAPP_CHAT_STORAGE=true` from this list): `WHATSAPP_AUTO_REJECT_CALL`, `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING`, `WHATSAPP_PRESENCE_ON_CONNECT`, `CHATWOOT_ENABLED`, `CHATWOOT_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`, `CHATWOOT_INBOX_ID`, `CHATWOOT_DEVICE_ID`, `CHATWOOT_IMPORT_MESSAGES`, `CHATWOOT_DAYS_LIMIT_IMPORT_MESSAGES` (+ `CHATWOOT_IMPORT_CONTACTS` per README only). `WHATSAPP_CHAT_STORAGE` is present on both sides (fork-side add by `git diff v8.1.2..main -- src/.env.example` confirms `+WHATSAPP_CHAT_STORAGE=true`).
- A but not B (`git diff v8.1.2..main -- src/.env.example` and `main:readme.md`): no net-new `WEBHOOK_*` / `CHATWOOT_*` env-var names on fork side. The fork's only env-line edit in this scope is extending `WHATSAPP_WEBHOOK_EVENTS` to append `history_sync_complete` to the comma-separated default list.

### Signature / auth code (HMAC, secret handling)

- `src/infrastructure/whatsapp/webhook.go`: unchanged in both directions vs `v8.1.2` (`git diff v8.1.2..upstream/main -- src/infrastructure/whatsapp/webhook.go` → empty; `git diff v8.1.2..main -- … ` → empty). The signing line `req.Header.Set("X-Hub-Signature-256", fmt.Sprintf("sha256=%s", signature))` at `webhook.go:48` is untouched on both sides.
- `src/pkg/utils/whatsapp.go` `GetMessageDigestOrSignature(msg, key []byte)` (`hmac.New(sha256.New, key)`): function signature unchanged in `main..upstream/main` diff (no `+`/`-` on the function header or `hmac.New` line); surrounding utility code does churn (the file accrues 11 upstream commits, 2 fork commits; shortstat for this file alone is included in the 96-file total).
- HMAC docs (`docs/webhook-payload.md` `### HMAC Signature Verification`, lines 75-117/79-121): identical between `main` and `upstream/main` (`diff <(git show main:…) <(git show upstream/main:…)` confirms no change in this block).
- `3b87f4e` subject names a "webhook auth fix" but `git show 3b87f4e -- src/pkg/utils/whatsapp.go` shows no `+`/`-` lines containing `hmac`, `signature`, `secret`, or `HMAC`. The commit body scopes the fix to chatwoot REST handlers (`src/ui/rest/chatwoot.go`, `src/cmd/rest.go` per commit body); the outgoing-webhook HMAC path in `webhook.go`/`whatsapp.go` is untouched. Precise mechanism of the "auth fix" is not extracted in this stage (needs decision).

### Webhook payload-struct fields (Go source)

- Files inspected: `src/infrastructure/whatsapp/webhook.go`, `webhook_forward.go`, `webhook_forward_test.go`.
- `git diff --stat main..upstream/main -- src/infrastructure/whatsapp/webhook_forward.go` → 1 file, +484 / −12; `webhook_forward_test.go` → +36 / 0; `webhook.go` → 0 / 0.
- Added in `webhook_forward.go` (upstream-side, per `git diff main..upstream/main`):
  - new imports: `hash/fnv`, `sync`, `time`, `github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/chatwoot`, `github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils`, `go.mau.fi/whatsmeow/types`.
  - new internal types/vars: `mutexShardCount` constant, `contactMutexShards` array, `groupNameCacheEntry` struct, `groupNameCache`/`groupNameCacheTTL`, `chatwootContactInfo` struct.
  - new functions: `getCachedGroupName`, `setCachedGroupName`, `getContactMutex`, `forwardToWebhooks`, `forwardToChatwoot`, `shouldForwardEventToChatwoot`, `isEventWhitelistedForChatwoot`, `extractChatwootContactInfo`, `buildChatwootMessageContent`, `buildReactionChatwootContent`, `extractStructuredMessageContent`.
  - Branch logic on event names: `case "message", "message.reaction":` (chatwoot allow-list), `case "message.reaction":`.
- No `json:"..."` struct-tag additions/removals in `webhook_forward.go`'s diff (the file uses `map[string]any` payloads, not typed structs).
- Top-level payload-map keys referenced as upstream string literals in `webhook_forward.go` (extracted via `grep -oE '"[a-z][a-z._]+"'` on `upstream/main:src/infrastructure/whatsapp/webhook_forward.go`): `audio`, `body`, `chat_id`, `contact`, `contacts_array`, `display_name`, `document`, `from`, `from_name`, `image`, `incoming`, `is_from_me`, `list`, `live_location`, `location`, `order`, `outgoing`, `phone_number`, `reacted_message_id`, `reaction`, `sticker`, `vcard`, `video`, `video_note`. The fork's `webhook_forward.go` references only `"context"`, `"fmt"`, `"strings"` — i.e., none of these payload-key tokens are present on fork side (`git show main:src/infrastructure/whatsapp/webhook_forward.go | grep -oE '"[a-z][a-z._]+"'`).

## Test coverage of the delta region

- Upstream tests touched (`v8.1.2..upstream/main`): **11 files** — `src/infrastructure/chatwoot/sync_test.go`, `src/infrastructure/whatsapp/chatwoot_forward_test.go`, `src/infrastructure/whatsapp/event_handler_test.go`, `src/infrastructure/whatsapp/event_message_test.go`, `src/infrastructure/whatsapp/webhook_forward_test.go`, `src/pkg/utils/environment_test.go`, `src/pkg/utils/general_test.go`, `src/pkg/utils/whatsapp_test.go`, `src/validations/message_validation_test.go`, `src/validations/send_validation_sticker_test.go`, `src/validations/send_validation_test.go`.
- Fork tests touched (`v8.1.2..main`): **3 files** — `src/infrastructure/whatsapp/jid_utils_test.go`, `src/pkg/utils/general_test.go`, `src/pkg/utils/whatsapp_test.go`.
- Overlap: **2 files** — `src/pkg/utils/general_test.go`, `src/pkg/utils/whatsapp_test.go`.

---

**Rules for this stage:**

- All facts. No "should we" or "I recommend".
- Cite commit SHAs for every claim.
- If a fact requires interpretation, mark it `(needs decision)` and leave it for D.
