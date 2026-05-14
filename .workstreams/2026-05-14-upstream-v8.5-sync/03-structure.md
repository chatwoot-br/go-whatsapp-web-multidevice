# S — Structure (~2 pages target)

QRSPI S stage. "Like a C header file: signatures and types, not implementation."

Produced from D. Constrains P to agreed interfaces — P cannot invent new signatures.

## File inventory

NEW upstream (arrive via reset, fresh in fork tree):

```
src/infrastructure/chatwoot/
  client.go                            # NEW upstream — REST client + echo-loop guard
  types.go                             # NEW upstream — Contact/Conversation/Message/WebhookPayload
  sync.go                              # NEW upstream — SyncService (history → Chatwoot)
  sync_types.go                        # NEW upstream — SyncProgress/SyncOptions/SyncRequest
  sync_test.go                         # NEW upstream
src/ui/rest/chatwoot.go                # NEW upstream — /chatwoot/webhook + /chatwoot/sync*
src/infrastructure/whatsapp/
  webhook_forward.go                   # MODIFIED upstream (+484/-12) + fork plug-in for history_sync_complete
  chatwoot_forward_test.go             # NEW upstream
docs/chatwoot.md                       # NEW upstream
```

NEW fork (re-applied as fresh commits on top of upstream):

```
src/pkg/utils/phone_br.go              # NEW fork — BR 9-digit layer on upstream phone.go (Q3)
src/pkg/utils/phone_br_test.go         # NEW fork — BR fixtures + jid_utils_test.go content folded in (OQ10)
src/infrastructure/whatsapp/forward_history_sync.go    # NEW fork (OQ3 default; in-place edit on webhook_forward.go is alternate)
```

MODIFIED (upstream rewrites the file; fork re-layers its delta):

```
src/pkg/utils/whatsapp.go              # MODIFIED — drop fork's ValidateAndNormalizeJID (moved to phone_br.go); upstream's ValidateJidWithLogin retained
src/pkg/utils/general.go               # MODIFIED — three-way merge (upstream 3 commits, fork 1)
src/pkg/utils/whatsapp_test.go         # MODIFIED — three-way merge
src/pkg/utils/general_test.go          # MODIFIED — three-way merge
src/infrastructure/whatsapp/history_sync.go            # MODIFIED — keep fork's deduplicateLIDChats + forwardHistorySyncCompleteToWebhook (OQ2 confirmed not subsumed)
src/infrastructure/whatsapp/jid_utils.go               # MODIFIED — keep NormalizeJIDFromLIDWithContext (OQ2)
src/infrastructure/chatstorage/sqlite_repository.go    # MODIFIED — keep fork's MergeLIDChat / GetLIDChats
src/infrastructure/chatstorage/device_repository.go    # MODIFIED — keep wrapper methods for above
src/domains/chatstorage/interfaces.go                  # MODIFIED — keep IChatStorageRepository.MergeLIDChat / GetLIDChats
src/usecase/send.go                    # MODIFIED — 22 BR-phone caller sites (per grep)
src/usecase/message.go                 # MODIFIED — 8 BR-phone caller sites
src/usecase/user.go                    # MODIFIED — 3 BR-phone caller sites
src/usecase/group.go                   # MODIFIED — 8 BR-phone caller sites
src/usecase/chat.go                    # MODIFIED — 3 BR-phone caller sites
src/usecase/newsletter.go              # MODIFIED — 1 BR-phone caller site
src/config/settings.go                 # MODIFIED — 9 CHATWOOT_* + WHATSAPP_AUTO_REJECT_CALL/PRESENCE_ON_CONNECT vars arrive; fork retains AppVersion bump path
src/cmd/root.go                        # MODIFIED — new CHATWOOT_* flag bindings
src/cmd/rest.go                        # MODIFIED — chatwoot routes registered; /chatwoot/webhook excluded from basic-auth (3b87f4e)
src/.env.example                       # MODIFIED — 8 CHATWOOT_* + 3 WHATSAPP_* added; fork appends history_sync_complete to WHATSAPP_WEBHOOK_EVENTS
src/go.mod / src/go.sum                # MODIFIED — whatsmeow + ~35 dep bumps
readme.md / docs/webhook-payload.md    # MODIFIED — event taxonomy + Chatwoot section
.github/workflows/build-docker-image.yaml              # MODIFIED — keep fork's `+`→`-` tag rewrite at :24
.github/workflows/release.yml                          # MODIFIED — fork-heavier; reconcile per OQ8
```

PRESERVED fork-only (untouched by reset):

```
charts/gowa/{Chart.yaml,values.yaml,templates/,README.md}   # PRESERVED fork — Helm chart
.github/workflows/chart-releaser.yaml                       # PRESERVED fork — Helm release
.github/workflows/set-latest-tag.yaml                       # PRESERVED fork — manual latest gate
docs/decisions/2026-01-18-fix-history-sync-lid-duplicate-chats.md   # PRESERVED fork
```

DELETED (fork delta retired by upstream):

```
src/pkg/utils/whatsapp.go::ValidateAndNormalizeJID     # DELETED — replaced by phone_br.go entry point
(no other deletions — fork did not delete files upstream still ships)
```

## Public-API signatures (no bodies)

```go
// src/pkg/utils/phone_br.go  (NEW fork — layered on upstream utils.NormalizePhoneE164 + utils.ParseJID)

// ValidateAndNormalizeJIDBR queries WhatsApp for the canonical JID, applying BR 9th-digit
// normalization (5566996679626 → 556696679626). Wraps upstream utils.NormalizePhoneE164 and
// utils.ParseJID; calls whatsmeow client.IsOnWhatsApp under a 10s context.Timeout.
// All v8.1.0+7 callers in src/usecase/{send,chat,group,message,newsletter,user}.go invoke this
// directly — contract `(client, jid) → (types.JID, error)` is preserved verbatim.
func ValidateAndNormalizeJID(client *whatsmeow.Client, jid string) (types.JID, error)

// Internal helper: phone-string normalization layered over upstream NormalizePhoneE164.
// Returned for unit-test reachability; not called outside the package.
func normalizePhoneBR(phone string) string

// src/pkg/utils/phone.go  (UPSTREAM — re-exported, fork does not modify)
func NormalizePhoneE164(phone string) string
func ExtractPhoneFromJID(jid string) string
func CleanPhoneForWhatsApp(phone string) string

// src/pkg/utils/whatsapp.go  (upstream signatures retained; fork's ValidateAndNormalizeJID removed)
func ValidateJidWithLogin(client *whatsmeow.Client, jid string) (types.JID, error)
func ResolveLIDToPhone(ctx context.Context, jid types.JID, client *whatsmeow.Client) types.JID
func ResolvePhoneToLID(ctx context.Context, jid types.JID, client *whatsmeow.Client) types.JID
func ParseJID(arg string) (types.JID, error)
func FormatJID(jid string) types.JID
func IsOnWhatsapp(client *whatsmeow.Client, jid string) bool
func GetMessageDigestOrSignature(msg, key []byte) (string, error)   // HMAC — unchanged on both sides

// src/infrastructure/chatwoot/types.go  (NEW upstream)
type Contact struct { ID int; Name, Email, PhoneNumber, Identifier string; CustomAttributes map[string]any }
type Conversation struct { ID, ContactID, InboxID int; Status string }
type Message struct { ID int; Content, MessageType string; Private bool; ContentType string }
type CreateContactRequest struct { InboxID int; Name, PhoneNumber, Identifier string; CustomAttributes map[string]any }
type CreateConversationRequest struct { InboxID, ContactID int; Status string }
type CreateMessageRequest struct { Content, MessageType string; Private bool }
type WebhookPayload struct {
    ID int; Event, MessageType, Content string; Private bool
    Account Account; Conversation ConversationWebhook; Sender Contact; Attachments []Attachment
}
type Attachment struct { ID int; FileType, DataURL string /* + upstream fields */ }
type ConversationWebhook struct { /* upstream fields */ }
type ConversationMeta struct { /* upstream fields */ }
type Account struct { /* upstream fields */ }

// src/infrastructure/chatwoot/client.go  (NEW upstream)
type Client struct { BaseURL, APIToken string; AccountID, InboxID int; HTTPClient *http.Client }
func NewClient() *Client
func GetDefaultClient() *Client
func MarkMessageAsSent(messageID int)
func IsMessageSentByUs(messageID int) bool
func (c *Client) IsConfigured() bool
func (c *Client) FindContactByIdentifier(identifier string, isGroup bool) (*Contact, error)
func (c *Client) CreateContact(name, identifier string, isGroup bool) (*Contact, error)
func (c *Client) FindOrCreateContact(name, identifier string, isGroup bool) (*Contact, error)
func (c *Client) UpdateContactName(contactID int, name string) error
func (c *Client) FindConversation(contactID int) (*Conversation, error)
func (c *Client) CreateConversation(contactID int) (*Conversation, error)
func (c *Client) FindOrCreateConversation(contactID int) (*Conversation, error)
func (c *Client) CreateMessage(conversationID int, content, messageType string, attachments []string) (int, error)

// src/infrastructure/chatwoot/sync_types.go  (NEW upstream)
type SyncState struct { /* db tags: device_id, chat_jid, last_synced_msg_id, last_synced_time, sync_status, messages_synced, messages_failed, created_at, updated_at */ }
type SyncProgress struct {
    DeviceID, Status string
    TotalChats, SyncedChats, FailedChats, TotalMessages, SyncedMessages, FailedMessages int
    CurrentChat string; StartedAt, CompletedAt *time.Time; Error string
    // mu sync.RWMutex (private)
}
type SyncOptions struct { DaysLimit int; IncludeMedia, IncludeGroups bool; MaxMessagesPerChat, BatchSize int; DelayBetweenBatches time.Duration }
type SyncRequest struct { DeviceID string; DaysLimit int; IncludeMedia, IncludeGroups bool }
type SyncResponse struct { Status, Message string; Progress *SyncProgress }
func DefaultSyncOptions() SyncOptions
func NewSyncProgress(deviceID string) *SyncProgress
func (p *SyncProgress) SetRunning()
func (p *SyncProgress) SetCompleted()
func (p *SyncProgress) SetFailed(err error)
func (p *SyncProgress) UpdateChat(chatJID string)
func (p *SyncProgress) IncrementSyncedChats()
func (p *SyncProgress) IncrementFailedChats()
func (p *SyncProgress) IncrementSyncedMessages()
func (p *SyncProgress) IncrementFailedMessages()
func (p *SyncProgress) SetTotals(chats, messages int)
func (p *SyncProgress) AddMessages(count int)
func (p *SyncProgress) Clone() SyncProgress
func (p *SyncProgress) IsRunning() bool

// src/infrastructure/chatwoot/sync.go  (NEW upstream)
type SyncService struct { /* client *Client; chatStorageRepo domainChatStorage.IChatStorageRepository; progressMap, progressMu (private) */ }
func NewSyncService(client *Client, chatStorageRepo domainChatStorage.IChatStorageRepository) *SyncService
func (s *SyncService) GetProgress(deviceID string) *SyncProgress
func (s *SyncService) IsRunning(deviceID string) bool
func (s *SyncService) SyncHistory(ctx context.Context, deviceID string, waClient *whatsmeow.Client, opts SyncOptions) (*SyncProgress, error)
func GetSyncService(client *Client, chatStorageRepo domainChatStorage.IChatStorageRepository) *SyncService
func GetDefaultSyncService() *SyncService

// src/ui/rest/chatwoot.go  (NEW upstream)
type ChatwootHandler struct { AppUsecase domainApp.IAppUsecase; SendUsecase domainSend.ISendUsecase; DeviceManager *whatsapp.DeviceManager; ChatStorageRepo domainChatStorage.IChatStorageRepository }
func NewChatwootHandler(appUsecase domainApp.IAppUsecase, sendUsecase domainSend.ISendUsecase, dm *whatsapp.DeviceManager, chatStorageRepo domainChatStorage.IChatStorageRepository) *ChatwootHandler
func (h *ChatwootHandler) HandleWebhook(c *fiber.Ctx) error
func (h *ChatwootHandler) SyncHistory(c *fiber.Ctx) error
func (h *ChatwootHandler) SyncStatus(c *fiber.Ctx) error

// src/infrastructure/whatsapp/history_sync.go  (MODIFIED fork — all three retained per OQ2)
func deduplicateLIDChats(ctx context.Context, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client, deviceID string)
func forwardHistorySyncCompleteToWebhook(ctx context.Context, client *whatsmeow.Client, syncType string)
// jid_utils.go helper — context-timeout variant kept; signature unchanged
func NormalizeJIDFromLIDWithContext(jid types.JID, client *whatsmeow.Client) types.JID
func NormalizeJIDFromLID(ctx context.Context, jid types.JID, client *whatsmeow.Client) types.JID

// src/domains/chatstorage/interfaces.go  (MODIFIED fork — keep additions on IChatStorageRepository)
type IChatStorageRepository interface {
    // ... upstream methods ...
    MergeLIDChat(deviceID, lidJID, phoneJID string) error
    GetLIDChats(deviceID string) ([]*Chat, error)
}

// src/infrastructure/whatsapp/webhook_forward.go  (MODIFIED upstream — fork plug-in)
// Upstream-native (unchanged contract; fork adopts as-is):
func forwardPayloadToConfiguredWebhooks(ctx context.Context, payload map[string]any, eventName string) error
func forwardToWebhooks(ctx context.Context, payload map[string]any, eventName string) error
func forwardToChatwoot(ctx context.Context, payload map[string]any, eventName string)
func shouldForwardEventToChatwoot(eventName string) bool
func isEventWhitelistedForChatwoot(eventName string) bool
func isEventWhitelisted(eventName string) bool

// The fork's history_sync_complete dispatch hooks in by calling forwardPayloadToConfiguredWebhooks
// (the same entry forwardHistorySyncCompleteToWebhook already uses in fork main). OQ3 decision
// deferred to P-time on WHERE the new event-name-aware branch lives:
//   Option A (default): new file src/infrastructure/whatsapp/forward_history_sync.go
//   Option B:           in-place edit on webhook_forward.go's dispatcher
// Either way the signature the plug-in must conform to is:
//   func forwardPayloadToConfiguredWebhooks(ctx context.Context, payload map[string]any, eventName string) error

// src/usecase/{send,chat,group,message,newsletter,user}.go  (MODIFIED — 45 call sites total)
// Every caller routes through the BR layered function; contract unchanged.
// Call shape at every site: utils.ValidateAndNormalizeJID(client, request.Phone) → (types.JID, error)
```

## Webhook contract surface

| Event | Status | Source / commit |
|---|---|---|
| `message` | EXISTING | both — `is_from_me` top-level addition from upstream `3b87f4e`; fork already populated this key in `event_message.go:82` so behaviorally no-op |
| `message.reaction` | EXISTING | both |
| `message.revoked` | EXISTING | both |
| `message.edited` | EXISTING | both |
| `message.ack` | EXISTING | both |
| `message.deleted` | EXISTING | both |
| `group.participants` | EXISTING | both |
| `group.joined` | EXISTING | both |
| `newsletter.joined` / `.left` / `.message` / `.mute` | EXISTING | both |
| `chat_presence` | NEW upstream | `c428afa` (typing/recording) |
| `call.offer` | NEW upstream | `5c193bc` (auto-reject feature) — only emitted event under `## Call Events`; no `call.terminate` ships in v8.5.0 per `docs/webhook-payload.md` |
| `history_sync_complete` | FORK-ADDED | retained; payload `{event, device_id, payload: {sync_type, timestamp}}` |

Field-level (top-level / payload-map keys post-upgrade):

| Key | Status | Notes |
|---|---|---|
| `is_from_me` (bool) | NEW upstream (`3b87f4e`) | Top-level under `### Common Payload Fields`. Fork already populates — behaviorally no-op. |
| `phone_number` on contact payload | NEW upstream (`437df12`) | Added under `### Contacts` |
| `contacts_array` | NEW upstream (`00ee65b`) | `ContactsArrayMessage` shape; `extractContactDetails` + `structuredContactsArraySummary` consume it |
| media `caption` in payload | UPSTREAM (`306391e`) | Fork already mirrors via auto-downloaded media path (v8.1.0+3) |
| Meta Ads referral keys (CTWA) | NEW upstream (`fe7d2c7`) | `ExtractExternalAdReply` adds `external_ad_reply` map |
| `chat_name` / `sender_name` | FORK-ADDED | Populated in `event_message.go:121` / fork v8.1.0+1; not in upstream payload doc — fork divergence persists |

## Configuration / env-var surface

Source-of-truth: `upstream/main:src/.env.example` (post-reset baseline) + fork's `phone_br.go` slice + fork's `WHATSAPP_WEBHOOK_EVENTS` extension.

| Env var | Default | Status | Notes |
|---|---|---|---|
| `APP_PORT` / `APP_HOST` / `APP_DEBUG` / `APP_OS` / `APP_BASIC_AUTH` / `APP_TRUSTED_PROXIES` | upstream defaults | UPSTREAM | unchanged |
| `APP_BASE_PATH` | `""` | UPSTREAM (also fork-retained) | present in both; fork-side middleware fix from v8.1.0+1 lives in fiber stack, not env-shape |
| `DB_URI` / `DB_KEYS_URI` | upstream defaults | UPSTREAM | unchanged |
| `WHATSAPP_AUTO_REPLY` | `"Auto reply message"` | UPSTREAM | unchanged |
| `WHATSAPP_AUTO_MARK_READ` | `false` | UPSTREAM | unchanged |
| `WHATSAPP_AUTO_REJECT_CALL` | `false` | NEW upstream (`b42727e`) | feature gate for `call.offer` event |
| `WHATSAPP_AUTO_DOWNLOAD_MEDIA` | `true` | UPSTREAM | unchanged |
| `WHATSAPP_WEBHOOK` | `https://webhook.site/...` | UPSTREAM | unchanged |
| `WHATSAPP_WEBHOOK_SECRET` | `super-secret-key` in `.env.example`; `secret` in `config/settings.go` | UPSTREAM | HMAC path untouched on both sides (R confirmed; OQ1 confirmed) |
| `WHATSAPP_WEBHOOK_INSECURE_SKIP_VERIFY` | `false` | UPSTREAM | unchanged |
| `WHATSAPP_WEBHOOK_EVENTS` | upstream default | UPSTREAM (FORK-EXTENDED) | fork appends `history_sync_complete` to the comma-separated list |
| `WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` | `false` | **DEPRECATED / DEAD** | Present in `upstream/main:src/.env.example` but no Go reference remains in `upstream/main` (only the literal `"outgoing"` appears in `webhook_forward.go` from `chatwootMessageTypeFromPayload`). Outgoing messages are now always forwarded; this var is a no-op. **Flag back to D: Investigation finding claimed `3b87f4e` removed the var — primary source disagrees, the var is still in `.env.example`. Behavior matches "always forwarded" (Investigation's payload claim), but the config knob is dead code, not deleted.** |
| `WHATSAPP_ACCOUNT_VALIDATION` | `true` | UPSTREAM | unchanged |
| `WHATSAPP_PRESENCE_ON_CONNECT` | `unavailable` | NEW upstream (`61c29b0`) | |
| `WHATSAPP_CHAT_STORAGE` | `true` | UPSTREAM (also fork-retained) | present in both sides |
| `CHATWOOT_ENABLED` | `false` | NEW upstream (`44a128c`) | |
| `CHATWOOT_URL` | `https://app.chatwoot.com` | NEW upstream (`44a128c`) | |
| `CHATWOOT_API_TOKEN` | `xxxxxxxx` | NEW upstream (`44a128c`) | |
| `CHATWOOT_ACCOUNT_ID` | `111111` | NEW upstream (`44a128c`) | |
| `CHATWOOT_INBOX_ID` | `000000` | NEW upstream (`44a128c`) | |
| `CHATWOOT_DEVICE_ID` | `""` | NEW upstream (`909b6e6`) | multi-device routing |
| `CHATWOOT_IMPORT_MESSAGES` | `false` | NEW upstream (`3b87f4e`) | history-sync gate |
| `CHATWOOT_DAYS_LIMIT_IMPORT_MESSAGES` | `3` | NEW upstream (`3b87f4e`) | |

**Count check**: 8 `CHATWOOT_*` vars in `upstream/main:src/.env.example`. The README and `3b87f4e` body mentioned a 9th (`CHATWOOT_IMPORT_CONTACTS`), but the same commit's "chore: remove unused ChatwootImportContacts config option" sub-commit deleted it before merge — not present in current `.env.example` or `readme.md`. **D's End-state count "9 new `CHATWOOT_*` env vars" is off by one — should read 8.** Flag back to D.

No fork-unique env vars exist beyond the `WHATSAPP_WEBHOOK_EVENTS` value extension. `charts/gowa/values.yaml` exposes upstream's 8 `CHATWOOT_*` knobs as Helm values in the same slice.

## Open at S-stage (signature decisions deferred to P)

- **OQ3 file boundary** for `history_sync_complete` plug-in (new file vs in-place). Signature is locked (`forwardPayloadToConfiguredWebhooks(ctx, payload, eventName) error`); location is not. P picks at code-time.
- **`phone_br.go` internal split** — whether `normalizePhoneBR` is unexported (default) or exported for fixture reuse. Decide at P; the exported `ValidateAndNormalizeJID` shape is locked.

## Flags back to D (codebase reads contradict D framing)

1. **`WHATSAPP_WEBHOOK_INCLUDE_OUTGOING` not removed.** Investigation finding claimed `3b87f4e` removed the var; primary source (`git show upstream/main:src/.env.example`) shows it still present. Behaviorally Investigation's "always forwarded" claim holds — no Go reference remains in `upstream/main`. Classify as DEPRECATED, not REMOVED. D should correct the End-state phrasing before P relies on it.
2. **Chatwoot env count off by one.** D End state says "9 new `CHATWOOT_*` env vars"; actual count is 8 (`CHATWOOT_IMPORT_CONTACTS` was added then removed inside `3b87f4e`).

---

**Rules:**

- Implementations forbidden. Bodies stay empty.
- If you can't write the signature, the design isn't done — go back to D.
- P will reference S; signature drift between S and P means S was incomplete.
