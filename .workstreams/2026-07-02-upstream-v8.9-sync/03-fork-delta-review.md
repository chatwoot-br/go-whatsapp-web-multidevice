# Fork-delta review vs upstream v8.9.0 — does each carried change still apply?

Raw diff: `fork-delta-vs-v8.9.0.diff` (this dir; `git diff upstream/v8.9.0 upgrade/v8.9.0-sync -- src docs readme.md .github`, 65 files, ~5,001+/374−). Three independent review passes (phone/send paths; whatsapp infra/event layer; chatwoot/config/CI), each verifying against `upstream/v8.9.0`, the 2-commit `upstream/main` tail, and whatsmeow `@20260622185415`.

**Verdict totals: 33 KEEP · 2 DROP-CANDIDATE · 3 already-converged (no delta left) · 2 KEEP-but-dormant.**

## Action items

1. **DROP-CANDIDATE (recommended revert): the LID caller swap.** The fork replaced `whatsapp.NormalizeJIDFromLID(ctx,…)` with `utils.ResolveLIDToPhone(ctx,…)` at ~10 call sites (`event_archive/chat_presence/delete/group/receipt/message.go`, `sqlite_repository.go`, `usecase/message.go`). Upstream's `utils.ResolveLIDToPhone` (`pkg/utils/whatsapp.go:696`) is **byte-identical** to the wrapper the fork deleted — zero behavior gain, and every swapped site is a file upstream actively edits, inflating future merge-conflict surface. `AGENTS.md:30` still prescribes the removed name. Reverting the call sites to upstream's wrapper while **keeping** the fork-only `NormalizeJIDFromLIDWithContext` (self-context resolver for post-debounce paths — genuinely needed) is behavior-neutral and shrinks the next sync's surface.
2. **Bookkeeping: HMAC is upstream-owned now.** `GetMessageDigestOrSignature`, the `X-Hub-Signature-256` header set (`webhook.go:48`), and the verifier (`ui/rest/chatwoot.go:114-119`) are byte-identical to upstream — none appear in the fork diff. The CHANGELOG/runbook "preserved fork feature" claim is stale; the fork carries only *tests* around it (keep those). Corrected in the v8.9.0+1 CHANGELOG entry.
3. **Dead-code pruning candidates (fork-internal, not upstream-related):** the info cache (`info_cache.go`, `pkg/cache/` — zero callers anywhere) and the ON_DEMAND request side (handler + device-props flags present; `BuildHistorySyncRequest` never called, so it only fires on rare unsolicited syncs). Both dormant-but-green; drop or wire deliberately.

## Already converged — upstream now owns these (nothing left to carry)

| Former fork concern | Where upstream has it |
|---|---|
| Webhook HMAC signing + verification | `pkg/utils/whatsapp.go`, `webhook.go:48`, `ui/rest/chatwoot.go:119` (byte-identical) |
| Receipt `Device==0` duplicate-webhook guard | `event_receipt.go:82-93` verbatim in v8.9.0 |
| Chatwoot token/URL trimming | upstream `client.go:138-139` |
| Chatwoot 1:1 name-preservation (#675/#688) | upstream `client.go:295` |
| `gowa_whatsapp_jid` write key | shared; only the *waha* read-fallback is fork-carried |

## KEEP verdicts (evidence-checked, no upstream equivalent)

### Phone normalization + send paths
| Change | Files | Evidence |
|---|---|---|
| BR ninth-digit stack: `normalizePhoneBR`, `brPhoneCandidates` (mobile-gated), `probeBRPhone`, probe outcome classification, bounded retry + total deadline, ambiguous fall-through, `resolveUserJID`/`ValidateAndNormalizeJID` | `pkg/utils/phone_br.go` + tests | Upstream has zero country-specific logic; its `IsOnWhatsapp`/`ValidateJidWithLogin` do one un-retried USync and hard-fail on miss |
| `IsOnWhatsapp` honest-ambiguity refactor (ambiguous ⇒ false) | `pkg/utils/whatsapp.go` | Upstream inline single-probe |
| Caller sweep to `ValidateAndNormalizeJID` (send×14, message×6, group×13, user×3, newsletter×1) | `usecase/*` | Upstream still calls `ValidateJidWithLogin` at all sites |
| `participantToJID`: connectivity guard (no MustLogin panic) + canonical-JID add + real error surfaced | `usecase/group.go` | Upstream `group.go:296-315` has neither |
| `DownloadImageFromURL` MIME→ext for extension-less S3 presigned URLs | `pkg/utils/general.go` | Upstream `general.go:367-387` still rejects extension-less URLs |
| `RevokeMessage` resolver call swap | `usecase/message.go` | Compile-forced by the LID rename — folds into action item 1 |

### WhatsApp infra / event layer
| Change | Files | Evidence |
|---|---|---|
| `history_sync_complete` debounced event (5s quiet) | `forward_history_sync.go`, `history_sync.go` | No upstream/whatsmeow completion event (only `HistorySync`, `OfflineSyncCompleted`) |
| LID chat dedup `deduplicateLIDChats` + `MergeLIDChat`/`GetLIDChats` (interface + wrappers + SQLite impl) | `history_sync.go`, `domains/chatstorage/`, `chatstorage_wrapper.go`, `sqlite_repository.go` | whatsmeow gives LID→PN mapping only; no chat-row dedup upstream |
| Push-name cache + 3-tier chat lookup | `history_sync.go` | Upstream single-JID lookup only |
| `RequireFullSync=true` on pairing (~1yr history) | `device_manager.go` | whatsmeow default false; upstream never sets it |
| `NormalizeJIDFromLIDWithContext` (fresh 30s ctx for post-debounce paths) | `jid_utils.go` | Fork-unique; event ctx is cancelled by debounce time |
| `InitWaDB` bounded retry + fail-fast + credential redaction (+ `root.go` Fatalf pair) | `database.go`, `cmd/root.go` | Upstream still panics once, no retry |
| WhatsApp-connection proxy (SOCKS5/HTTP/HTTPS) + `ProxyIP` surfacing (incl. DeviceManager.js) | `init.go`, `device_manager.go`, `device_instance.go`, `domains/device`, views | No proxy vars upstream (only HTTP `AppTrustedProxies`) |
| `chat_name` webhook field for outgoing msgs (storage→contact lookup) + `sender_name` / `SenderName` on `MessageInfo` | `event_message*.go`, `domains/chat/chat.go`, `usecase/chat.go` | Upstream has only `from_name` (pushname) |

### Chatwoot / config / CI / docs
| Change | Files | Evidence |
|---|---|---|
| `waha_whatsapp_jid` read fallback (pre-rebrand contacts keep routing) | `infrastructure/chatwoot/client.go`, `ui/rest/chatwoot.go` | Zero `waha` hits upstream |
| Fiber `UnescapePath` + `restFiberConfig()` + test | `cmd/rest.go`, `cmd/rest_test.go` | Not set upstream. **More critical post-#740**: upstream's absent-chat fix turns the encoded-JID miss from loud-500 into a silent 0-message import |
| Webhook events default adds `chat_presence,call.offer,history_sync_complete` | `.env.example` | Upstream default lacks all three |
| e2e suite (HMAC round-trip, history_sync_complete, chatwoot REST, LID dedup, taxonomy) | `src/e2e/integration_test.go` | Fork-only harness |
| CI identity: ghcr multi-arch, `vX.Y.Z+N` tag enforcement, ghcr set-latest, Helm chart-releaser | `.github/workflows/*` | Fork registry/versioning; no upstream conflict |
| Fork docs (upstream-sync runbook, decisions, plans, webhook-payload fork sections, readme rows) | `docs/`, `readme.md` | Fork process/feature docs |
| `chatwoot/sync.go` + `sync_rest_test.go` | — | **No fork delta** — identical to upstream |

### whatsmeow bump obsoletes nothing
The 20260622 whatsmeow supplies primitives the fork already uses (`GetPNForLID`, `SetProxyAddress`, `RequireFullSync`/`OnDemandReady` fields) but none of the app-level policy the fork carries (dedup, completion signaling, retry, probe classification, webhook shaping). The `IsOnWhatsApp` single-USync behavior the probe stack guards against is unchanged.
