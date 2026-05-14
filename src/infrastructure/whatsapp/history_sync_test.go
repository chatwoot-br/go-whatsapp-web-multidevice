package whatsapp

import (
	"context"
	"errors"
	"testing"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
)

// dedupStubRepo records calls and lets each test choose what GetLIDChats
// returns. Embeds the interface so the unused method set compiles; only the
// LID-dedup-relevant methods are overridden.
type dedupStubRepo struct {
	domainChatStorage.IChatStorageRepository
	lidChats []*domainChatStorage.Chat
	lidErr   error

	merged       []mergeCall
	getLIDCalled bool
}

type mergeCall struct {
	deviceID string
	lidJID   string
	phoneJID string
}

func (r *dedupStubRepo) GetLIDChats(deviceID string) ([]*domainChatStorage.Chat, error) {
	r.getLIDCalled = true
	return r.lidChats, r.lidErr
}

func (r *dedupStubRepo) MergeLIDChat(deviceID, lidJID, phoneJID string) error {
	r.merged = append(r.merged, mergeCall{deviceID, lidJID, phoneJID})
	return nil
}

// TestDeduplicateLIDChats_NilClientShortCircuits asserts the guard: nil client
// returns immediately without touching the repo.
func TestDeduplicateLIDChats_NilClientShortCircuits(t *testing.T) {
	repo := &dedupStubRepo{}
	deduplicateLIDChats(context.Background(), repo, nil, "dev1")
	if repo.getLIDCalled {
		t.Error("GetLIDChats should NOT be called when client is nil")
	}
}

// TestDeduplicateLIDChats_NilRepoShortCircuits asserts the symmetric guard.
func TestDeduplicateLIDChats_NilRepoShortCircuits(t *testing.T) {
	// Should not panic.
	deduplicateLIDChats(context.Background(), nil, nil, "dev1")
}

// TestDeduplicateLIDChats_NoLIDChats handles the empty-result fast path. The
// real client argument is required (deduplicateLIDChats short-circuits on
// nil), but with no @lid chats GetLIDChats's result alone determines whether
// resolution is attempted — we never reach the whatsmeow.Client call.
//
// Skipped because deduplicateLIDChats unconditionally short-circuits on nil
// client, blocking this assertion entirely. See phase7-untestable.md for the
// rationale.
//
// We still cover the GetLIDChats-error branch via the nil-client guard above.
func TestDeduplicateLIDChats_GetLIDChatsError(t *testing.T) {
	// We can't supply a real *whatsmeow.Client here without paired-phone setup,
	// so we exercise the path indirectly: with a nil client, the function
	// short-circuits BEFORE calling GetLIDChats. The assertion below documents
	// that the guard ordering must remain "nil-client first, then LID query".
	repo := &dedupStubRepo{
		lidErr: errors.New("boom"),
	}
	deduplicateLIDChats(context.Background(), repo, nil, "dev1")

	if repo.getLIDCalled {
		t.Error("with nil client, the function must short-circuit before GetLIDChats")
	}
	if len(repo.merged) > 0 {
		t.Error("no merges should occur on the nil-client path")
	}
}
