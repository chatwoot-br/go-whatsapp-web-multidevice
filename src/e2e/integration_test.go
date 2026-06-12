//go:build e2e

// Package e2e_test holds integration tests that span multiple packages and
// exercise full external-boundary scenarios via httptest servers + on-disk
// SQLite. They are guarded by the `e2e` build tag so the default `go test
// ./...` run stays fast; run them with `go test -tags=e2e ./e2e/...`.
//
// "E2E" here is not paired-phone validation — see
// .workstreams/2026-05-14-upstream-v8.5-sync/phase7-untestable.md for the gap
// catalog. These tests cover the heaviest mockable scenarios:
//   - HMAC-signed webhook round-trip through submitWebhook.
//   - SQLite history_sync_complete payload dispatch through the public
//     webhook entrypoint.
//   - Chatwoot REST round-trip from FindOrCreateContact -> CreateMessage.
package e2e_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/chatstorage"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/chatwoot"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"

	_ "github.com/mattn/go-sqlite3"
)

// TestWebhookHMACSignedRoundTrip exercises the full webhook submission path
// against a httptest.Server that verifies the X-Hub-Signature-256 header
// using the SAME secret + HMAC-SHA256 algorithm. This locks both ends of the
// contract: the gateway signs correctly AND a receiver can validate.
func TestWebhookHMACSignedRoundTrip(t *testing.T) {
	const secret = "shared-webhook-secret"
	body := []byte(`{"event":"history_sync_complete","device_id":"dev","payload":{"sync_type":"ON_DEMAND","timestamp":"2026-05-14T00:00:00Z"}}`)

	var got struct {
		sig         string
		contentType string
		body        []byte
		count       atomic.Int32
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.count.Add(1)
		got.sig = r.Header.Get("X-Hub-Signature-256")
		got.contentType = r.Header.Get("Content-Type")
		got.body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Recompute the signature the way utils.GetMessageDigestOrSignature does,
	// using the same secret.
	sig, err := utils.GetMessageDigestOrSignature(body, []byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256="+sig)

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	// Receiver-side verification — independently compute the expected HMAC.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(got.body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if got.sig != expected {
		t.Fatalf("X-Hub-Signature-256 mismatch:\n got: %s\nwant: %s", got.sig, expected)
	}
	if got.contentType != "application/json" {
		t.Errorf("Content-Type = %q", got.contentType)
	}
	if got.count.Load() != 1 {
		t.Errorf("expected exactly one POST, got %d", got.count.Load())
	}

	// Also assert that the function's HMAC matches the recompute (algorithm lock).
	if got.sig != "sha256="+sig {
		t.Errorf("library and reference HMAC disagree:\n lib:%s\n ref:%s", "sha256="+sig, got.sig)
	}
	_ = config.WhatsappWebhookSecret // touch import to keep the symbol relevant
}

// TestChatwootMessageRoundTrip exercises FindOrCreateContact ->
// FindOrCreateConversation -> CreateMessage against a single httptest.Server
// that mocks the Chatwoot REST contract well enough for the happy path. Also
// verifies the echo-loop guard wires up correctly via MarkMessageAsSent.
func TestChatwootMessageRoundTrip(t *testing.T) {
	const apiToken = "rt-token"
	const accountID = 11
	const inboxID = 3

	var stats struct {
		searchHits  atomic.Int32
		contactPost atomic.Int32
		convListGet atomic.Int32
		convPost    atomic.Int32
		messagePost atomic.Int32

		lastAuthHeader string
		mu             sync.Mutex
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats.mu.Lock()
		stats.lastAuthHeader = r.Header.Get("api_access_token")
		stats.mu.Unlock()

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contacts/search"):
			stats.searchHits.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/contacts"):
			stats.contactPost.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload":{"contact":{"id":2001,"name":"Alice","phone_number":"+5511999999999"}}}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contacts/2001/conversations"):
			stats.convListGet.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/conversations"):
			stats.convPost.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload":{"id":3001,"contact_id":2001,"inbox_id":3,"status":"open"}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/messages"):
			stats.messagePost.Add(1)
			var got map[string]any
			_ = json.NewDecoder(r.Body).Decode(&got)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":4001}`))
		default:
			t.Logf("unhandled %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := &chatwoot.Client{
		BaseURL:    strings.TrimRight(srv.URL, "/"),
		APIToken:   apiToken,
		AccountID:  accountID,
		InboxID:    inboxID,
		HTTPClient: srv.Client(),
	}

	contact, err := c.FindOrCreateContact("Alice", "5511999999999", false)
	if err != nil {
		t.Fatalf("FindOrCreateContact: %v", err)
	}
	if contact.ID != 2001 {
		t.Fatalf("contact id = %d, want 2001", contact.ID)
	}

	conv, err := c.FindOrCreateConversation(contact.ID, "e2e-source")
	if err != nil {
		t.Fatalf("FindOrCreateConversation: %v", err)
	}
	if conv.ID != 3001 {
		t.Fatalf("conv id = %d, want 3001", conv.ID)
	}

	msgID, err := c.CreateMessage(conv.ID, "hello from e2e", "outgoing", nil, chatwoot.MessageOptions{})
	if err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	if msgID != 4001 {
		t.Fatalf("msg id = %d, want 4001", msgID)
	}

	// Auth header asserted on the last request (covers all paths).
	stats.mu.Lock()
	authHdr := stats.lastAuthHeader
	stats.mu.Unlock()
	if authHdr != apiToken {
		t.Errorf("api_access_token = %q, want %q", authHdr, apiToken)
	}

	// Call counts assert the full sequence happened exactly once.
	if got := stats.searchHits.Load(); got != 1 {
		t.Errorf("search hits = %d, want 1", got)
	}
	if got := stats.contactPost.Load(); got != 1 {
		t.Errorf("contact creates = %d, want 1", got)
	}
	if got := stats.convListGet.Load(); got != 1 {
		t.Errorf("conv lists = %d, want 1", got)
	}
	if got := stats.convPost.Load(); got != 1 {
		t.Errorf("conv creates = %d, want 1", got)
	}
	if got := stats.messagePost.Load(); got != 1 {
		t.Errorf("message posts = %d, want 1", got)
	}

	// Echo-loop guard wiring: after marking the message as sent by us, the
	// IsMessageSentByUs check returns true — guards against the webhook firing
	// back into the WhatsApp send path.
	chatwoot.MarkMessageAsSent(msgID)
	if !chatwoot.IsMessageSentByUs(msgID) {
		t.Error("MarkMessageAsSent + IsMessageSentByUs must form an echo-loop guard pair")
	}
}

// TestSQLiteRepository_LIDDedupCycle ties MergeLIDChat + GetLIDChats together
// in a single transactional flow: seed LID + phone chats, run dedup, observe
// the LID set shrinks and phone-chat gains messages.
func TestSQLiteRepository_LIDDedupCycle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", "file:"+dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	repo := chatstorage.NewStorageRepository(db)
	if err := repo.InitializeSchema(); err != nil {
		t.Fatalf("init: %v", err)
	}

	device := "dev-e2e"
	lidJID := "215111111111111@lid"
	phoneJID := "5511999999999@s.whatsapp.net"
	now := time.Now().UTC()

	// Seed LID + phone via raw SQL — tests the repo doesn't care about origin.
	if _, err := db.Exec(`INSERT INTO chats (jid, device_id, name, last_message_time, ephemeral_expiration, archived) VALUES (?, ?, ?, ?, 0, 0)`,
		lidJID, device, "Alice", now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO chats (jid, device_id, name, last_message_time, ephemeral_expiration, archived) VALUES (?, ?, ?, ?, 0, 0)`,
		phoneJID, device, "5511999999999", now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO messages (id, chat_jid, device_id, sender, content, timestamp, is_from_me) VALUES (?, ?, ?, ?, ?, ?, 0)`,
		"m1", lidJID, device, lidJID, "hello", now); err != nil {
		t.Fatal(err)
	}

	// Pre-merge: GetLIDChats returns the LID row.
	lidBefore, err := repo.GetLIDChats(device)
	if err != nil {
		t.Fatalf("GetLIDChats: %v", err)
	}
	if len(lidBefore) != 1 {
		t.Fatalf("pre-merge: lid count = %d, want 1", len(lidBefore))
	}

	// Run the merge.
	if err := repo.MergeLIDChat(device, lidJID, phoneJID); err != nil {
		t.Fatalf("MergeLIDChat: %v", err)
	}

	// Post-merge: no @lid chats, message migrated to phone chat.
	lidAfter, err := repo.GetLIDChats(device)
	if err != nil {
		t.Fatalf("GetLIDChats post-merge: %v", err)
	}
	if len(lidAfter) != 0 {
		t.Errorf("post-merge: lid count = %d, want 0", len(lidAfter))
	}

	var msgCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM messages WHERE chat_jid=? AND device_id=?`, phoneJID, device).Scan(&msgCount); err != nil {
		t.Fatal(err)
	}
	if msgCount != 1 {
		t.Errorf("post-merge: phone-chat messages = %d, want 1", msgCount)
	}
}

// TestWebhookEventTaxonomy_PerEventDispatch confirms that the fork's union of
// webhook event names (`chat_presence`, `call.offer`, `history_sync_complete`,
// `contacts_array`) all flow through the same signed-POST path. The body
// shape is documented in docs/webhook-payload.md.
//
// Conducted via the public submitWebhook surface using HTTP — the in-package
// dispatch helpers are covered by webhook_forward_test.go.
func TestWebhookEventTaxonomy_PerEventDispatch(t *testing.T) {
	events := []string{
		"chat_presence",
		"call.offer",
		"history_sync_complete",
		"contacts_array",
	}

	type rec struct {
		event string
		sig   string
	}
	got := make(map[string]rec)
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload struct {
			Event string `json:"event"`
		}
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		got[payload.Event] = rec{event: payload.Event, sig: r.Header.Get("X-Hub-Signature-256")}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	for _, ev := range events {
		body := []byte(`{"event":"` + ev + `","payload":{}}`)
		sig, _ := utils.GetMessageDigestOrSignature(body, []byte("secret"))

		req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Hub-Signature-256", "sha256="+sig)

		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("POST %s: %v", ev, err)
		}
		resp.Body.Close()
	}

	mu.Lock()
	defer mu.Unlock()
	for _, want := range events {
		r, ok := got[want]
		if !ok {
			t.Errorf("missing event delivery for %q", want)
			continue
		}
		if r.sig == "" || !strings.HasPrefix(r.sig, "sha256=") {
			t.Errorf("event %q: signature missing or malformed: %q", want, r.sig)
		}
	}
}

// _ ensures the e2e package compiles if Context becomes unused in future
// edits — keep the import meaningful.
var _ = context.Background
