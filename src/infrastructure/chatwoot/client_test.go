package chatwoot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// testClient wires a Client to a httptest.Server's URL and matching token.
// Constructed directly (not via GetDefaultClient) to side-step the sync.Once
// singleton — every test gets a fresh client whose BaseURL points to its own
// httptest.Server.
func testClient(srv *httptest.Server) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(srv.URL, "/"),
		APIToken:   "test-token",
		AccountID:  42,
		InboxID:    7,
		HTTPClient: srv.Client(),
	}
}

// TestIsConfigured lives in client_methods_test.go (upstream's variant covers
// the same IsConfigured() contract plus the all-empty case).

// TestCreateContact_AuthTokenHeader asserts that the api_access_token header is
// sent on outbound requests — the fork's auth contract with Chatwoot.
func TestCreateContact_AuthTokenHeader(t *testing.T) {
	var gotToken string
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("api_access_token")
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"payload":{"contact":{"id":111,"name":"Alice"}}}`))
	}))
	defer srv.Close()

	c := testClient(srv)
	contact, err := c.CreateContact("Alice", "5511999999999", false)
	if err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	if contact == nil || contact.ID != 111 {
		t.Fatalf("contact = %+v, want ID 111", contact)
	}
	if gotToken != "test-token" {
		t.Errorf("api_access_token = %q, want test-token", gotToken)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
}

// TestCreateContact_GroupUsesIdentifier verifies the routing logic: groups
// post the identifier (group JID) and omit phone_number, so Chatwoot can
// distinguish them from human contacts.
func TestCreateContact_GroupUsesIdentifier(t *testing.T) {
	var captured CreateContactRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"payload":{"contact":{"id":200,"name":"My Group"}}}`))
	}))
	defer srv.Close()

	c := testClient(srv)
	groupJID := "120363111111111@g.us"
	if _, err := c.CreateContact("My Group", groupJID, true); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	if captured.Identifier != groupJID {
		t.Errorf("identifier = %q, want %q", captured.Identifier, groupJID)
	}
	if captured.PhoneNumber != "" {
		t.Errorf("phone_number = %q, want empty for groups", captured.PhoneNumber)
	}
	// New contacts are written with upstream's gowa_whatsapp_jid key (the fork
	// reads waha_whatsapp_jid as a back-compat fallback for pre-rebrand data).
	if got := captured.CustomAttributes["gowa_whatsapp_jid"]; got != groupJID {
		t.Errorf("custom_attributes.gowa_whatsapp_jid = %v, want %v", got, groupJID)
	}
}

// TestCreateContact_LIDUsesIdentifier asserts the fork's @lid handling path:
// non-phone identifiers must go to Identifier, NOT phone_number.
func TestCreateContact_LIDUsesIdentifier(t *testing.T) {
	var captured CreateContactRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"payload":{"contact":{"id":300}}}`))
	}))
	defer srv.Close()

	c := testClient(srv)
	lidJID := "215946727821336@lid"
	if _, err := c.CreateContact("Anon", lidJID, false); err != nil {
		t.Fatalf("CreateContact: %v", err)
	}
	if captured.Identifier != lidJID {
		t.Errorf("identifier = %q, want %q", captured.Identifier, lidJID)
	}
	if captured.PhoneNumber != "" {
		t.Errorf("phone_number = %q, want empty for @lid identifier", captured.PhoneNumber)
	}
}

// TestFindContactByIdentifier_PhoneSearchPath ensures the phone-number case
// normalizes to E.164 before issuing the search query.
func TestFindContactByIdentifier_PhoneSearchPath(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		// Return an empty payload — we only care about the query shape here.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"payload":[]}`))
	}))
	defer srv.Close()

	c := testClient(srv)
	if _, err := c.FindContactByIdentifier("5511999999999", false); err != nil {
		t.Fatalf("FindContactByIdentifier: %v", err)
	}
	if !strings.HasPrefix(gotQuery, "+") {
		t.Errorf("query = %q, want E.164 (leading +)", gotQuery)
	}
}

// TestCreateMessage_PostsToConversationsEndpoint asserts the URL shape upstream
// expects: /api/v1/accounts/<id>/conversations/<conv_id>/messages.
func TestCreateMessage_PostsToConversationsEndpoint(t *testing.T) {
	var gotPath string
	var gotBody CreateMessageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":999}`))
	}))
	defer srv.Close()

	c := testClient(srv)
	id, err := c.CreateMessage(123, "hello", "incoming", nil, MessageOptions{})
	if err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	if id != 999 {
		t.Errorf("id = %d, want 999", id)
	}
	wantPath := "/api/v1/accounts/42/conversations/123/messages"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody.Content != "hello" || gotBody.MessageType != "incoming" {
		t.Errorf("body = %+v", gotBody)
	}
}

// TestMarkMessageAsSent_IsMessageSentByUs_EchoLoopGuard asserts the echo-loop
// guard: a message marked sent by us is recognized; a fresh ID is not. This
// is the core invariant preventing WhatsApp -> Chatwoot -> WhatsApp loops.
func TestMarkMessageAsSent_IsMessageSentByUs_EchoLoopGuard(t *testing.T) {
	// Use a sufficiently large ID to avoid collisions with any other test
	// running in this package (sentMessageIDs is package-level state).
	id := 999_888_777

	if IsMessageSentByUs(id) {
		t.Fatalf("precondition: id %d should be unknown", id)
	}
	MarkMessageAsSent(id)
	if !IsMessageSentByUs(id) {
		t.Fatalf("id %d should be flagged as sent by us after MarkMessageAsSent", id)
	}
	// Calling again must keep returning true (no delete-on-check; multiple
	// webhook events for the same message are expected).
	if !IsMessageSentByUs(id) {
		t.Fatalf("id %d should still be flagged on second check", id)
	}
}

// TestMarkMessageAsSent_ZeroIDIgnored documents the noop guard.
func TestMarkMessageAsSent_ZeroIDIgnored(t *testing.T) {
	MarkMessageAsSent(0)
	if IsMessageSentByUs(0) {
		t.Fatal("zero ID should never be tracked")
	}
}

// TestIsMessageSentByUs_TTLExpiry uses reflection-free temporary state to
// simulate TTL expiry. The function reads `time.Since(storedAt) > TTL` and
// returns false after expiry; we overwrite the stored time directly.
func TestIsMessageSentByUs_TTLExpiry(t *testing.T) {
	id := 888_777_666

	// Manually store an expired entry.
	sentMessageIDs.Store(id, time.Now().Add(-2*sentMessageIDsTTL))

	if IsMessageSentByUs(id) {
		t.Fatal("expired entry should not be reported as sent by us")
	}
	// The expired entry is also cleaned up by the check itself.
	if _, ok := sentMessageIDs.Load(id); ok {
		t.Error("expired entry should be deleted after a check")
	}
}

// TestCreateContact_BadStatusError verifies that non-2xx responses surface as
// errors (not swallowed) — Chatwoot 5xx should not be hidden.
func TestCreateContact_BadStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := testClient(srv)
	_, err := c.CreateContact("X", "5511000000000", false)
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

// Name-update-on-find behavior is now covered by upstream's
// TestFindOrCreateContact_{PreservesExistingIndividualName,FillsBlankExistingIndividualName,RefreshesExistingGroupName}
// in client_methods_test.go (the fork adopts upstream's preserve-existing-1:1-name
// semantics; the old always-overwrite test was dropped).

// TestClient_ConcurrentSafety lightly stress-tests the package-level
// sentMessageIDs map under concurrent MarkMessageAsSent / IsMessageSentByUs
// to confirm there's no data race (run with -race).
func TestClient_ConcurrentSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency stress in -short mode")
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			MarkMessageAsSent(1_000_000 + id)
			_ = IsMessageSentByUs(1_000_000 + id)
		}(i)
	}
	wg.Wait()
}

func TestFindOrCreateContact_PreservesExistingIndividualName(t *testing.T) {
	tests := []struct {
		name         string
		incomingName string
	}{
		{
			name:         "incoming WhatsApp name differs",
			incomingName: "Alice WA",
		},
		{
			name:         "incoming name is phone fallback",
			incomingName: "6281234567890",
		},
		{
			name:         "incoming name is empty",
			incomingName: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			putCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/1/contacts/search":
					if got := r.URL.Query().Get("q"); got != "+6281234567890" {
						t.Fatalf("search q = %q, want +6281234567890", got)
					}
					writeJSON(t, w, http.StatusOK, map[string]any{
						"payload": []Contact{{
							ID:          123,
							Name:        "Manual Alice",
							PhoneNumber: "+6281234567890",
						}},
					})
				case r.Method == http.MethodPut && r.URL.Path == "/api/v1/accounts/1/contacts/123":
					putCalls++
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
			}))
			defer server.Close()

			client := &Client{
				BaseURL:    server.URL,
				APIToken:   "token",
				AccountID:  1,
				InboxID:    2,
				HTTPClient: server.Client(),
			}

			contact, err := client.FindOrCreateContact(tc.incomingName, "6281234567890", false)
			if err != nil {
				t.Fatalf("FindOrCreateContact: %v", err)
			}
			if contact.Name != "Manual Alice" {
				t.Fatalf("contact name = %q, want Manual Alice", contact.Name)
			}
			if putCalls != 0 {
				t.Fatalf("PUT contact calls = %d, want 0", putCalls)
			}
		})
	}
}

func TestFindOrCreateContact_FillsBlankExistingIndividualName(t *testing.T) {
	putCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/1/contacts/search":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"payload": []Contact{{
					ID:          123,
					Name:        "",
					PhoneNumber: "+6281234567890",
				}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/accounts/1/contacts/123":
			putCalls++
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode PUT body: %v", err)
			}
			if body["name"] != "6281234567890" {
				t.Fatalf("PUT name = %q, want 6281234567890", body["name"])
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		APIToken:   "token",
		AccountID:  1,
		InboxID:    2,
		HTTPClient: server.Client(),
	}

	contact, err := client.FindOrCreateContact("6281234567890", "6281234567890", false)
	if err != nil {
		t.Fatalf("FindOrCreateContact: %v", err)
	}
	if contact.Name != "6281234567890" {
		t.Fatalf("contact name = %q, want 6281234567890", contact.Name)
	}
	if putCalls != 1 {
		t.Fatalf("PUT contact calls = %d, want 1", putCalls)
	}
}

func TestFindOrCreateContact_RefreshesExistingGroupName(t *testing.T) {
	const groupJID = "120363123456789@g.us"
	putCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/1/contacts/search":
			if got := r.URL.Query().Get("q"); got != groupJID {
				t.Fatalf("search q = %q, want %s", got, groupJID)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"payload": []Contact{{
					ID:         456,
					Name:       "Old Group",
					Identifier: groupJID,
				}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/accounts/1/contacts/456":
			putCalls++
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode PUT body: %v", err)
			}
			if body["name"] != "New Group" {
				t.Fatalf("PUT name = %q, want New Group", body["name"])
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		APIToken:   "token",
		AccountID:  1,
		InboxID:    2,
		HTTPClient: server.Client(),
	}

	contact, err := client.FindOrCreateContact("New Group", groupJID, true)
	if err != nil {
		t.Fatalf("FindOrCreateContact: %v", err)
	}
	if contact.Name != "New Group" {
		t.Fatalf("contact name = %q, want New Group", contact.Name)
	}
	if putCalls != 1 {
		t.Fatalf("PUT contact calls = %d, want 1", putCalls)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("write JSON: %v", err)
	}
}
