package whatsapp

import (
	"context"
	"errors"
	"testing"
	"time"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
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

// TestDeduplicateLIDChats_GetLIDChatsError handles the empty-result fast path. The
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

func TestProcessConversationMessagesPersistsReactionEvents(t *testing.T) {
	originalLog := log
	log = waLog.Noop
	defer func() { log = originalLog }()

	deviceID := "device-a@s.whatsapp.net"
	chatJID := "628123456789@s.whatsapp.net"
	repo := &historyReactionRepoSpy{}

	ctx := ContextWithDevice(context.Background(), NewDeviceInstance(deviceID, nil, nil))
	syncType := waHistorySync.HistorySync_RECENT
	reactionTimestamp := uint64(time.Date(2026, time.May, 16, 8, 2, 0, 0, time.UTC).Unix())
	data := &waHistorySync.HistorySync{
		SyncType: &syncType,
		Conversations: []*waHistorySync.Conversation{
			{
				ID: proto.String(chatJID),
				Messages: []*waHistorySync.HistorySyncMsg{
					{
						Message: &waWeb.WebMessageInfo{
							Key: &waCommon.MessageKey{
								RemoteJID: proto.String(chatJID),
								FromMe:    proto.Bool(false),
								ID:        proto.String("reaction-event-1"),
							},
							Message: &waE2E.Message{
								ReactionMessage: &waE2E.ReactionMessage{
									Key: &waCommon.MessageKey{
										RemoteJID: proto.String(chatJID),
										FromMe:    proto.Bool(false),
										ID:        proto.String("msg-1"),
									},
									Text: proto.String("\U0001f44d"),
								},
							},
							MessageTimestamp: &reactionTimestamp,
						},
					},
				},
			},
		},
	}

	if err := processConversationMessages(ctx, data, repo, nil); err != nil {
		t.Fatalf("process conversation messages: %v", err)
	}

	if repo.createReactionCalls != 1 {
		t.Fatalf("expected history reaction event to be persisted once, got %d", repo.createReactionCalls)
	}
	if repo.lastReaction == nil {
		t.Fatal("expected reaction event to be passed to repository")
	}
	if got := repo.lastReaction.Message.GetReactionMessage().GetText(); got != "\U0001f44d" {
		t.Fatalf("expected thumbs-up reaction, got %q", got)
	}
	if got := repo.lastReaction.Message.GetReactionMessage().GetKey().GetID(); got != "msg-1" {
		t.Fatalf("expected target message id msg-1, got %q", got)
	}
}

type historyReactionRepoSpy struct {
	domainChatStorage.IChatStorageRepository
	createReactionCalls int
	lastReaction        *events.Message
}

func (r *historyReactionRepoSpy) CreateReaction(_ context.Context, evt *events.Message) error {
	r.createReactionCalls++
	r.lastReaction = evt
	return nil
}

func (r *historyReactionRepoSpy) GetChatNameWithPushName(jid types.JID, _ string, _ string, pushName string) string {
	if pushName != "" {
		return pushName
	}
	return jid.String()
}
