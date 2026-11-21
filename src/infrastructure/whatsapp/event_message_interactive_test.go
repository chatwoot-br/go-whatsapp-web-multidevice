package whatsapp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func interactiveTestEvent(id string, fromMe bool, msg *waE2E.Message) *events.Message {
	bot := types.NewJID("552120180792", types.DefaultUserServer)
	me := types.NewJID("5566999991670", types.DefaultUserServer)
	sender := bot
	if fromMe {
		sender = me
	}
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     bot,
				Sender:   sender,
				IsFromMe: fromMe,
			},
			ID:        id,
			Timestamp: time.Date(2026, time.November, 19, 12, 49, 3, 0, time.UTC),
		},
		Message: msg,
	}
}

// TestBuildEventPayloadDT3Sequence replays the DT3 Cadeiras exchange that
// Chatwoot dropped: a bot text, the bot's list menu, and the tenant's pick
// echoed from WhatsApp Web. Every step must reach the webhook with a body.
func TestBuildEventPayloadDT3Sequence(t *testing.T) {
	listMessage := &waE2E.ListMessage{
		Description: proto.String("Escolha uma das opções"),
		ButtonText:  proto.String("Ver opções"),
		ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
		Sections: []*waE2E.ListMessage_Section{{
			Rows: []*waE2E.ListMessage_Row{
				{Title: proto.String("Pedidos"), RowID: proto.String("1")},
				{Title: proto.String("Dúvidas"), RowID: proto.String("2")},
				{Title: proto.String("Compras Corporativas"), RowID: proto.String("3")},
			},
		}},
	}

	steps := []struct {
		name     string
		evt      *events.Message
		wantBody string
		wantList bool
	}{
		{
			name:     "bot text",
			evt:      interactiveTestEvent("3A0001", false, &waE2E.Message{Conversation: proto.String("Sobre qual assunto você deseja falar?")}),
			wantBody: "Sobre qual assunto você deseja falar?",
		},
		{
			name:     "bot list menu",
			evt:      interactiveTestEvent("3A0002", false, &waE2E.Message{ListMessage: listMessage}),
			wantBody: "Escolha uma das opções\n\n[Ver opções]\n1. Pedidos\n2. Dúvidas\n3. Compras Corporativas",
			wantList: true,
		},
		{
			name: "tenant pick echoed from WhatsApp Web",
			evt: interactiveTestEvent("3EB029AA", true, &waE2E.Message{ListResponseMessage: &waE2E.ListResponseMessage{
				Title:             proto.String("Pedidos"),
				ListType:          waE2E.ListResponseMessage_SINGLE_SELECT.Enum(),
				SingleSelectReply: &waE2E.ListResponseMessage_SingleSelectReply{SelectedRowID: proto.String("1")},
				ContextInfo:       &waE2E.ContextInfo{StanzaID: proto.String("3A0002")},
			}}),
			wantBody: "Pedidos",
		},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			eventType, payload, err := buildEventPayload(context.Background(), nil, step.evt, nil)
			require.NoError(t, err)
			assert.Equal(t, EventTypeMessage, eventType)
			assert.Equal(t, step.wantBody, payload["body"])

			list, hasList := payload["list"]
			assert.Equal(t, step.wantList, hasList)
			if step.wantList {
				assert.Same(t, listMessage, list, "raw list proto must still be forwarded")
			}
		})
	}
}

func TestBuildEventPayloadRendersMenusAndPicksAsBody(t *testing.T) {
	tests := []struct {
		name     string
		fromMe   bool
		msg      *waE2E.Message
		wantBody string
	}{
		{
			name: "buttons message",
			msg: &waE2E.Message{ButtonsMessage: &waE2E.ButtonsMessage{
				ContentText: proto.String("Confirma?"),
				Buttons: []*waE2E.ButtonsMessage_Button{
					{ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{DisplayText: proto.String("Sim")}},
					{ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{DisplayText: proto.String("Não")}},
				},
			}},
			wantBody: "Confirma?\n\n1. Sim\n2. Não",
		},
		{
			name: "notification template",
			msg: &waE2E.Message{TemplateMessage: &waE2E.TemplateMessage{
				Format: &waE2E.TemplateMessage_HydratedFourRowTemplate_{
					HydratedFourRowTemplate: &waE2E.TemplateMessage_HydratedFourRowTemplate{
						HydratedContentText: proto.String("Seu pedido foi entregue."),
					},
				},
			}},
			wantBody: "Seu pedido foi entregue.",
		},
		{
			name: "interactive cta_url",
			msg: &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{
				Body: &waE2E.InteractiveMessage_Body{Text: proto.String("Veja o catálogo")},
				InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
					NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
						Buttons: []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{{
							Name:             proto.String("cta_url"),
							ButtonParamsJSON: proto.String(`{"display_text":"Abrir","url":"https://example.com"}`),
						}},
					},
				},
			}},
			wantBody: "Veja o catálogo\n\n1. Abrir: https://example.com",
		},
		{
			name:   "buttons response echo",
			fromMe: true,
			msg: &waE2E.Message{ButtonsResponseMessage: &waE2E.ButtonsResponseMessage{
				Response:         &waE2E.ButtonsResponseMessage_SelectedDisplayText{SelectedDisplayText: "Sim"},
				SelectedButtonID: proto.String("sim"),
			}},
			wantBody: "Sim",
		},
		{
			name:   "template button reply echo",
			fromMe: true,
			msg: &waE2E.Message{TemplateButtonReplyMessage: &waE2E.TemplateButtonReplyMessage{
				SelectedID:          proto.String("ok"),
				SelectedDisplayText: proto.String("Recebi"),
			}},
			wantBody: "Recebi",
		},
		{
			name:   "interactive response echo",
			fromMe: true,
			msg: &waE2E.Message{InteractiveResponseMessage: &waE2E.InteractiveResponseMessage{
				Body: &waE2E.InteractiveResponseMessage_Body{Text: proto.String("Abrir")},
			}},
			wantBody: "Abrir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, payload, err := buildEventPayload(context.Background(), nil, interactiveTestEvent("ID", tt.fromMe, tt.msg), nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantBody, payload["body"])
		})
	}
}

// TestBuildChatwootMessageContentUsesRenderedMenu checks that the built-in
// Chatwoot forwarder shows the rendered menu instead of the old "List: ..."
// summary or the unsupported placeholder.
func TestBuildChatwootMessageContentUsesRenderedMenu(t *testing.T) {
	_, payload, err := buildEventPayload(context.Background(), nil, interactiveTestEvent("ID", false, &waE2E.Message{
		ButtonsMessage: &waE2E.ButtonsMessage{
			ContentText: proto.String("Confirma?"),
			Buttons: []*waE2E.ButtonsMessage_Button{
				{ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{DisplayText: proto.String("Sim")}},
			},
		},
	}), nil)
	require.NoError(t, err)

	content, attachments := buildChatwootMessageContent(payload, false, "")
	assert.Equal(t, "Confirma?\n\n1. Sim", content)
	assert.Empty(t, attachments)
}
