# 02 — Chatwoot / webhook contract-drift recheck (Phase B)

Date: 2026-06-11. Scope: does merging upstream's tail (`upstream/v8.7.0..upstream/main`,
base now **v8.8.0**) drift the fork's **standard webhook output** (`forwardToWebhooks`) or
its **Chatwoot client** away from what `chatwoot-app` / existing production data expect?
Model: `.workstreams/2026-05-14-upstream-v8.5-sync/11-contract-drift.md`.

Refs: gowa = this repo (branch `upgrade/v8.7.0-sync`); cw = `code/chatwoot-app`.

---

## A. Webhook payload contract (gowa → chatwoot-app `Webhooks::WhatsappWebController`)

chatwoot-app reads only top-level `device_id` / `event` / `payload`, switches on known
`event` values, and **graceful-ignores unknown keys + unknown events** (established in the
v8.5 `11-contract-drift.md`; chatwoot-app is not part of this merge — unchanged).

| Item | Class | Evidence |
|---|---|---|
| **`session_id`** (new, upstream `1316f70`) | **ADDITIVE-SAFE** | New **top-level** key added by `addWebhookSessionID` (`webhook_forward.go:124`) only when `device_id`'s JID maps to a registered session; omitted otherwise. chatwoot-app ignores unknown top-level keys → no parser impact. |
| Fork fields `chat_name` / `sender_name` | **STABLE** | Still emitted (`event_message.go`). |
| `history_sync_complete` event | **STABLE** | Still emitted (`forward_history_sync.go`, `history_sync.go`); documented in `docs/webhook-payload.md` (event list kept on merge). |
| HMAC `X-Hub-Signature-256` | **STABLE** | `webhook.go:42,48` — `GetMessageDigestOrSignature(postBody, secret)` over raw body, header `sha256=<hex>`. Unchanged. |
| Upstream additive events/fields (reactions, message edits) | **ADDITIVE-SAFE** | New `message.reaction` payloads + edit fields; chatwoot-app branches on known events, ignores rest. |

**Verdict A: 0 breaking, 1 additive (`session_id`), HMAC stable.** No chatwoot-app code change required.

---

## B. Chatwoot client reconciliation (the real divergence this merge introduced)

Upstream evolved the **shared** `infrastructure/chatwoot/` (the fork's *active* integration,
not the dormant native module). Two changes needed a decision:

### B1. Custom-attribute rebrand `waha_whatsapp_jid` → `gowa_whatsapp_jid`

Upstream renamed the contact custom attribute that stores the WhatsApp JID. The auto-merge
took `gowa_whatsapp_jid` everywhere (`client.go`, inbound `ui/rest/chatwoot.go`, new `pgimport/`).

- **Cross-repo check (B5):** `git grep waha_whatsapp_jid|gowa_whatsapp_jid` in `chatwoot-app`,
  `chatwoot-operator`, `gitops` → **clean** (no external reader). Attribute is GoWA-internal
  (written on create, read for find + inbound agent-reply routing).
- **Risk:** existing fork-production Chatwoot contacts are stored under `waha_whatsapp_jid`.
  `FindOrCreateContact` matches them by `Identifier`/phone and **early-returns**, so
  `CreateContact` never re-runs → those contacts are **never rewritten to `gowa`**. A naive
  "adopt gowa" would permanently break inbound routing (agent replies) for every pre-existing
  contact — no self-heal.
- **DECISION — write `gowa`, read both.** New writes use `gowa_whatsapp_jid` (follow upstream,
  minimize rebase-surface). Read paths fall back to the legacy key:
  - `client.go` `FindContactByIdentifier`: match `gowa_whatsapp_jid`, then `waha_whatsapp_jid`.
  - `ui/rest/chatwoot.go` inbound route: destination from `gowa_whatsapp_jid`, else `waha_whatsapp_jid`, else phone.
  Fork-only divergence is ~6 lines; existing data keeps resolving indefinitely.
- **Open (lower priority):** the new `pgimport/` direct-Postgres path queries/writes `gowa` only
  (`writer.go`). If the fork enables `CHATWOOT_IMPORT_DB_URI` against an instance holding legacy
  `waha` contacts, import-time matching could create duplicates. Add a `waha` fallback to the
  pgimport upsert SQL **iff** direct-DB import is used on legacy data. (Import path off by default.)

### B2. Contact-name handling: overwrite → preserve

Upstream (`9266165` #688, `23a8de2` #675) changed `FindOrCreateContact` to **preserve an
existing non-empty 1:1 name** (only fill blank 1:1 names, refresh group names) instead of the
fork's prior **always-overwrite-on-find**.

- **DECISION — adopt upstream's preserve semantics** (it is a bug fix: stops clobbering an
  agent-edited / saved contact name with a WhatsApp pushname or phone number). Behavior change
  owned here, not silent. Dropped the fork's obsolete `TestFindOrCreateContact_UpdatesNameOnFind`;
  kept upstream's `Preserves/FillsBlank/RefreshesGroup` tests.
- **Behavioral note:** WhatsApp-side 1:1 pushname changes no longer propagate to an
  already-named Chatwoot contact (group subjects still do). Acceptable for the fork.

---

## C. Human-owned gates (real-gateway UAT, before merge to origin/main)

- Paired-phone + live Chatwoot round-trip: confirm inbound agent reply routes for **both** a
  **new** contact (`gowa_whatsapp_jid`) and a **pre-existing** contact (`waha_whatsapp_jid` via
  fallback). This validates the B1 decision on real data.
- Confirm `session_id` appears on webhooks for session-mapped devices and is benign in chatwoot-app.
- Confirm name-preserve (B2) matches operator expectations on a real conversation.

**Overall: 0 breaking changes to the chatwoot-app contract.** The fork-internal client divergence
(B1 read-both, B2 name semantics) is contained in-repo and covered by automated tests; the only
residual items are UAT confirmations and the optional pgimport fallback.
