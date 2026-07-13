package whatsapp

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func msgEventForTest(msg *waE2E.Message) *events.Message {
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:   types.NewJID("628123450099", types.DefaultUserServer),
				Sender: types.NewJID("628111111111", types.DefaultUserServer),
			},
			ID:        "classify-1",
			Timestamp: time.Date(2026, time.July, 12, 8, 0, 0, 0, time.UTC),
		},
		Message: msg,
	}
}

// ClassifyMessageEvent is the pre-payload gate's view of what a message WILL publish as, so
// it must agree with what buildEventPayload actually publishes. If the two ever drift, the
// gate decides on one event name while delivery uses another — admitting media downloads it
// meant to block, or (worse) dropping events it meant to keep. This pins them together.
func TestClassifyMessageEvent_AgreesWithBuildEventPayload(t *testing.T) {
	revoke := waE2E.ProtocolMessage_REVOKE
	edit := waE2E.ProtocolMessage_MESSAGE_EDIT

	cases := []struct {
		name string
		msg  *waE2E.Message
		want string
	}{
		{"plain text", &waE2E.Message{Conversation: protoString("hello")}, EventTypeMessage},
		{"reaction", &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{Text: protoString("👍")}}, EventTypeMessageReaction},
		{"revoke", &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{Type: &revoke}}, EventTypeMessageRevoked},
		{"edit", &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{Type: &edit}}, EventTypeMessageEdited},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			evt := msgEventForTest(tc.msg)

			got := ClassifyMessageEvent(evt)
			if got != tc.want {
				t.Fatalf("ClassifyMessageEvent = %q, want %q", got, tc.want)
			}

			// The builder must reach the same verdict (nil client/repo: these paths do no
			// media or storage work — which is exactly why classification is safe pre-payload).
			built, _, err := buildEventPayload(context.Background(), nil, evt, nil)
			if err != nil {
				t.Fatalf("buildEventPayload: %v", err)
			}
			if built != got {
				t.Fatalf("classifier and payload builder disagree: gate would use %q, delivery publishes %q", got, built)
			}
		})
	}
}
