# Post-merge audit (§7.5) + fork-delta review — v8.10.0 sync

Raw fork-delta: `fork-delta-vs-v8.10.0.diff` (this dir; `git diff upstream/v8.10.0 upgrade/v8.10.0-sync -- src docs readme.md .github`, 54 files, +3969/−271 — down from 65 files/+5001 at v8.9.0: upstream absorbed more, fork carried less).

## 1. Device-scoping sweep — CLEAN

`git grep -nE '\.(GetChat|GetMessageByID|GetChatMessageCount)\(' src/usecase` → 3 hits
(`message.go:87`, `message.go:143`, `send.go:1820`), all **byte-identical to origin/main**
(pre-merge fork tip; reviewed in the v8.9 audit). The merge changed neither file.
New upstream usecase code: per-device webhook CRUD keys everything by explicit `deviceID`
(scoped by construction); `newsletter.GetMessages` reads only from whatsmeow (no DB);
`app.go` passkey flows touch no chat storage.

## 2. Default-flip diff — CLEAN

`git diff origin/main HEAD -- src/.env.example src/config/settings.go src/pkg/sqlite/` →
only `AppVersion v8.9.0+1 → v8.10.0+1` (ours) and a new `WhatsappTypeNewsletter` constant.
`ChatStorageMaxOpenConns = 1` intact with its invariant comment. `.env.example` webhook-events
default still carries `chat_presence,call.offer,history_sync_complete`.

## 3. whatsmeow-workaround check — CLEAN

- `normalizeLIDBounded` (30s per-call LID deadline): 10 uses in `history_sync.go`, intact.
- `InitWaDB` bounded retry + fail-fast: intact (`database.go`).
- Receipt `Device == 0` duplicate guard: intact (`event_receipt.go:82-93`).
- Event-handler context detach: still in the shared dispatcher (`event_handler.go:33`,
  `context.WithoutCancel`); upstream's new passkey events ride the same dispatcher.
- Fork features verified present: `phone_br.go`, `forward_history_sync.go`,
  `RequireFullSync` (device_manager.go ×2), proxy wiring (init/device_manager/device_instance/
  config/cmd), `chat_name`/`sender_name` (`event_message.go` 4-arg chain).

## Merge-resolution decisions (the 5 conflicts)

| File | Resolution |
|---|---|
| `chatstorage_wrapper.go` | Kept **both** appended method sets: fork `MergeLIDChat`/`GetLIDChats` + upstream per-device webhook delegates |
| `device_instance.go` | Kept both imports (fork `golang.org/x/net/proxy` + upstream `whatsmeow/types` for passkey fields) |
| `event_message_handler.go` | Adopted upstream structure (broadcast early-return; unconditional forward — per-device webhooks decide downstream) while keeping the fork's 4-arg `forwardMessageToWebhook(…, repo)` for the `chat_name` lookup |
| `config/settings.go` | `AppVersion = "v8.10.0+1"` (release rail) |
| `docs/webhook-payload.md` | Event-list union: upstream's `label.edit`/`label.association` + fork's `history_sync_complete` |

Post-merge adaptations (commit f00dee8): upstream tests updated to fork signatures
(`handleWebhookForward` 4-arg; `submitWebhookFn` stubs +`DeviceWebhookConfig`);
`newsletter.GetMessages` swapped `ValidateJidWithLogin` → `ValidateAndNormalizeJID`
per the fork's caller-sweep convention (only remaining upstream-name reference is a comment).

## Notable upstream semantics adopted (deliberate, not fallout)

- **#728 keep-slot logout** (Hub Operacional — our own upstreamed fix): remote logout now
  disconnects and keeps the device slot + chat history; the fork's old
  `TruncateAllDataWithLogging("REMOTE_LOGOUT")` is gone **by design**. Orphaned rows of a
  re-paired slot are scoped by the old JID (no cross-device leak; storage-only cost).
- **Per-device webhooks (#671)**: fleet deployments are unaffected until a device row sets
  `webhook_url` (see `02-contract-drift.md`).

## 4. xhigh code review — 10 finder angles → 8 verifier agents → sweep

~40 candidates, deduped to ~21 correctness + ~16 cleanup clusters; every correctness cluster
adversarially verified (CONFIRMED/PLAUSIBLE/REFUTED with file:line evidence).

**Fixed on-branch (commit 65f6c24, fork-relevant or unambiguous):**
1. `AddDevice` keyed webhook config by requested id, not `inst.ID()` — auto-id create 500s + orphans the slot.
2. Per-event `GetDeviceRecordByJID` on the `MaxOpenConns=1` pool → 30s TTL cache (nil results cached,
   invalidated on webhook writes, bypassed under test seam). The one finding that hits fleet directly.
3. `deleteStoreRowsForJID` broke after the first NonAD match — stale same-account store rows survived
   logout and were resurrected by `LoadExistingDevices` on restart.
4. `PurgeDevice` lacked the stale-id JID fallback `keepSlotLogout` has — `DELETE /devices/{jid}` on a
   uuid-keyed slot silently left store rows and the live slot (old pre-merge code's direct AD-JID match
   had covered this).
5. `MustLogin` moved before the non-user-JID early return in `ValidateAndNormalizeJID` — disconnected
   clients on group/newsletter endpoints now 401 (upstream parity) instead of raw 500. (The reviewed
   f00dee8 newsletter swap was consistent-with-fork but had inherited this fork-wide gap.)

**CONFIRMED, reported upstream-only (per-device webhook users; fleet unaffected):**
PATCH-as-PUT wipes omitted secret/events; URL-less config accepted+echoed but inert; TLS skip
OR-merge (device false can't restore verification; docs say "override"); empty device secret signs
tenant URLs with the global secret; empty device events list falls to global whitelist (docs: "all
events"); MCP mode never drains `websocket.Broadcast` (pre-existing; passkey adds 3 send sites; ~5min
handler-queue stall per event + leaked goroutine); UI passkey confirm posts to selected device, not
broadcast `device_id`; ungated payload build downloads all media types to PathMedia for discarded
payloads when no consumer configured (webhook path is the ONLY downloader of non-image media);
logout of unknown device 500s (was 200 no-op); `jid=""` never re-persisted for JID-keyed slots after
keep-slot logout (webhook config silently unbinds; restart does NOT repair JID-keyed slots);
`GetDeviceRecord`/`ListDeviceRecords` don't hydrate webhook columns (latent);
`SaveDeviceRecord` writes only display_name/jid (orphan-match save at device_manager.go:456 clobbers
display_name); fallback-to-global on lookup error + device-URL-replaces-global are DOCUMENTED
upstream design (operational footguns for the fleet — noted in 02-contract-drift.md).

**PLAUSIBLE:** PasskeyResponse clear-state race (needs a stall > full network RTT).
**REFUTED:** `webhookStorageForTest` data race (suite is sequential; all sends synchronized).

**Sweep pass (fresh finder over the fixed branch) — 4 gaps, 2 fixed (commit 55fa753):**
- FIXED: the PurgeDevice fix's `deviceID = inst.ID()` reassignment made `DeleteDeviceData` use the
  slot id only; chat/message rows are NonAD-JID-keyed → now deleted under both keys.
- FIXED: MCP server had no `WithRecovery()` — usecase MustLogin panics (incl. the widened gate from
  the phone_br.go fix) killed the whole process from an MCP tool goroutine; REST-only Recovery.
- REPORTED: 4 new MCP send tools (video/document/audio/poll) skip `SanitizePhone` — bare group ids
  parse as user JIDs (`ui/mcp/send.go`).
- REPORTED: `GetDeviceRecordByJID` `WHERE jid=? LIMIT 1` with no `ORDER BY` — duplicate same-JID
  device rows can resolve to the config-less row (per-device webhook silently ignored).

**Codex adversarial review (post-PR, verdict needs-attention) — all 4 high findings fixed (4866f27):**
1. Logout-then-DELETE orphaned retained JID-scoped history → `devices.last_jid` (migration 35),
   purge deletes under slot id + jid + last_jid; record getters unified on one column list
   (also resolves the hydration-divergence REPORTED item above).
2. No-consumer media downloads → `hasAnyWebhookConsumer` gate before payload build
   (global ∨ Chatwoot ∨ TTL-cached device scan) + early-exit in forwardPayloadToConfiguredWebhooks.
3. Lookup-error fallback to global → serve last-known cached config (even expired) first;
   global fallback only with no cached knowledge (kept for fleet availability — deliberate
   middle ground vs hard fail-closed).
4. Purge on partial failure destroyed the retry key → slot + record kept on error;
   DeleteDeviceRecord errors surfaced.

**Issue #10 (remote-logout registry=1/store=0)** — largely resolved by upstream #728 in this sync;
completed by: callback-independent keep-slot cleanup in handleLoggedOut, typed 401
`ErrSessionDeleted` from Reconnect/ReconnectDevice (re-pair guidance instead of raw 500),
auto-connect "awaiting re-pair" info log. `-race` clean on the whatsapp package.

**Cleanup (reported, not applied — fork-surface discipline: all in upstream-owned code):**
dead URL-only webhook API chain (usecase `SetDeviceWebhook`/`GetDeviceWebhook` + repo URL variants,
zero callers); duplicated whitelist match loop; duplicated logout flow (app vs device usecase);
MCP handler preamble copy-paste (~100 lines); `getDeviceRecordForTest` test-naming on the prod path;
"Forwarding to 0 webhooks" Info spam; `GetDeviceWebhook` IIFE nil-guards; scattered JID-class filters.

---

## Codex review round 2 (PR #11, head `f6590e0c` → `fcb565d`)

Codex's second pass raised **15 threads, all P2**. The load-bearing question for a *sync* PR is
not "is the finding true" but **"is the code ours"** — a bug that arrives verbatim with the
upstream tag is an upstream bug, and patching it here grows exactly the rebase surface §1 exists
to protect. Classification method (per finding): `git diff upstream/v8.10.0..HEAD -- <file>` for
file-level provenance, then `git blame` on the flagged line, then `git merge-base --is-ancestor
<blamed-sha> upstream/v8.10.0` to settle authorship.

**Result: 6 fork-authored (fixed), 1 already fixed, 8 pure upstream (deferred).**

### Fixed — fork-authored (all introduced by this sync's own fix commits, `65f6c24d` / `4866f27f`)

| Finding | Fix |
|---|---|
| `hasAnyWebhookConsumer` is **fleet-wide**: one device's per-device webhook admitted *every* device into payload construction (→ media downloaded, then discarded) | `hasWebhookConsumerForDevice(deviceJID)` — global ∨ Chatwoot ∨ *this* device's webhook; fails **open** on lookup error |
| Keep-slot logout cleared the in-memory JID even when store cleanup **failed** → retry deleted nothing, reported success, orphan row resurrected by `LoadExistingDevices` | retry recovers the identity from `last_jid` (Codex's own second option) |
| `last_jid` holds **one** identity: logout(A) → re-pair(B) → logout(B) overwrote the only pointer to A's retained chat data | `purgeSupersededRetainedJID` drops A's data as the pointer is replaced; same-JID (retry) stays a no-op |
| `ResetClient` nils the client, so a logged-out slot reached `ReconnectDevice` as a **nil client** — the `ErrSessionDeleted` branch added for it was unreachable behind a generic 500 | nil client on an existing slot → `ErrSessionDeleted` |
| `POST /devices` left the slot registered when webhook persistence failed → same `device_id` then failed "device already exists"; create unretryable | roll the slot back via `RemoveDevice` on both failure paths |
| `GetDeviceRecordByJID`: `LIMIT 1` with no `ORDER BY` could return a legacy config-less duplicate row over the named slot carrying the webhook | order by configured-first, then `updated_at DESC` |

Each has a regression test; the ordering one was **verified failing without the fix** (returns the
legacy row). Gates re-run green at `fcb565d` (`logs/gate1-3`).

Already fixed before this round: the round-1 "no-consumer media download" finding — `4866f27f`
added the pre-payload gate; round 2 only sharpened its scope (row 1 above).

### Deferred — pure upstream v8.10.0 code (replied + resolved on the PR, tracked in CHANGELOG)

Passkey UI confirm (skip-handoff **and** wrong-device confirm) and the passkey `websocket.Broadcast`
block in MCP mode (upstream #754, `Aldino Kemal`); `PATCH /devices/:id/webhook` PUT semantics,
device webhook TLS-override, empty device secret, empty device event-list (upstream #671,
`Mohamed Habib`); MCP send tools missing `SanitizePhone`.

All eight are faithful to upstream v8.10.0 and **unreachable in this fork's production topology**:
we pair by QR (not passkey), deliver through **one global webhook** per tenant (not per-device), and
run **REST** mode (not MCP). Fixing them here would put fork edits into upstream-pristine files for
features we do not run. They stay in CHANGELOG's *Known upstream issues* and are forwardable upstream.
