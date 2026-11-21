package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func nativeFlowButton(name, params string) *waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton {
	return &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
		Name:             proto.String(name),
		ButtonParamsJSON: proto.String(params),
	}
}

func nativeFlowInteractive(body string, buttons ...*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton) *waE2E.InteractiveMessage {
	return &waE2E.InteractiveMessage{
		Body: &waE2E.InteractiveMessage_Body{Text: proto.String(body)},
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
				Buttons:        buttons,
				MessageVersion: proto.Int32(1),
			},
		},
	}
}

func hydratedQuickReply(text, id string) *waE2E.HydratedTemplateButton {
	return &waE2E.HydratedTemplateButton{
		HydratedButton: &waE2E.HydratedTemplateButton_QuickReplyButton{
			QuickReplyButton: &waE2E.HydratedTemplateButton_HydratedQuickReplyButton{
				DisplayText: proto.String(text),
				ID:          proto.String(id),
			},
		},
	}
}

func hydratedURLButton(text, url string) *waE2E.HydratedTemplateButton {
	return &waE2E.HydratedTemplateButton{
		HydratedButton: &waE2E.HydratedTemplateButton_UrlButton{
			UrlButton: &waE2E.HydratedTemplateButton_HydratedURLButton{
				DisplayText: proto.String(text),
				URL:         proto.String(url),
			},
		},
	}
}

// dt3ListMessage mirrors the menu the DT3 Cadeiras bot sent: a single-select
// list opened by "Ver opções".
func dt3ListMessage() *waE2E.ListMessage {
	return &waE2E.ListMessage{
		Title:       proto.String(""),
		Description: proto.String("Escolha uma das opções"),
		ButtonText:  proto.String("Ver opções"),
		ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
		Sections: []*waE2E.ListMessage_Section{{
			Title: proto.String("Assuntos"),
			Rows: []*waE2E.ListMessage_Row{
				{Title: proto.String("Pedidos"), RowID: proto.String("1")},
				{Title: proto.String("Dúvidas"), RowID: proto.String("2")},
				{Title: proto.String("Compras Corporativas"), Description: proto.String("Para empresas"), RowID: proto.String("3")},
			},
		}},
	}
}

func TestExtractInteractiveText(t *testing.T) {
	tests := []struct {
		name string
		msg  *waE2E.Message
		want string
	}{
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
		{
			name: "plain text is not interactive",
			msg:  &waE2E.Message{Conversation: proto.String("oi")},
			want: "",
		},
		{
			name: "list message (DT3 menu)",
			msg:  &waE2E.Message{ListMessage: dt3ListMessage()},
			want: "Escolha uma das opções\n\n" +
				"[Ver opções]\nAssuntos\n1. Pedidos\n2. Dúvidas\n3. Compras Corporativas — Para empresas",
		},
		{
			name: "list message numbers rows in one sequence across sections, footer in head",
			msg: &waE2E.Message{ListMessage: &waE2E.ListMessage{
				Title:       proto.String("Atendimento"),
				Description: proto.String("Selecione"),
				FooterText:  proto.String("Responda com o número"),
				ButtonText:  proto.String("Menu"),
				Sections: []*waE2E.ListMessage_Section{
					{Title: proto.String("Vendas"), Rows: []*waE2E.ListMessage_Row{{Title: proto.String("Orçamento")}, {Title: proto.String("Pedido")}}},
					{Title: proto.String("Suporte"), Rows: []*waE2E.ListMessage_Row{{Title: proto.String("Falar com atendente")}}},
				},
			}},
			want: "Atendimento\nSelecione\nResponda com o número\n\n" +
				"[Menu]\nVendas\n1. Orçamento\n2. Pedido\n\nSuporte\n3. Falar com atendente",
		},
		{
			name: "buttons message with text header",
			msg: &waE2E.Message{ButtonsMessage: &waE2E.ButtonsMessage{
				Header:      &waE2E.ButtonsMessage_Text{Text: "Confirmação"},
				HeaderType:  waE2E.ButtonsMessage_TEXT.Enum(),
				ContentText: proto.String("Deseja confirmar o agendamento?"),
				FooterText:  proto.String("Clínica Exemplo"),
				Buttons: []*waE2E.ButtonsMessage_Button{
					{ButtonID: proto.String("sim"), ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{DisplayText: proto.String("Sim")}, Type: waE2E.ButtonsMessage_Button_RESPONSE.Enum()},
					{ButtonID: proto.String("nao"), ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{DisplayText: proto.String("Não")}, Type: waE2E.ButtonsMessage_Button_RESPONSE.Enum()},
				},
			}},
			want: "Confirmação\nDeseja confirmar o agendamento?\nClínica Exemplo\n\n1. Sim\n2. Não",
		},
		{
			name: "buttons message with image header renders text only",
			msg: &waE2E.Message{ButtonsMessage: &waE2E.ButtonsMessage{
				Header:      &waE2E.ButtonsMessage_ImageMessage{ImageMessage: &waE2E.ImageMessage{Mimetype: proto.String("image/jpeg")}},
				HeaderType:  waE2E.ButtonsMessage_IMAGE.Enum(),
				ContentText: proto.String("Promoção da semana"),
				Buttons: []*waE2E.ButtonsMessage_Button{
					{ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{DisplayText: proto.String("Quero")}, Type: waE2E.ButtonsMessage_Button_RESPONSE.Enum()},
				},
			}},
			want: "Promoção da semana\n\n1. Quero",
		},
		{
			name: "hydrated template with quick replies and URL button (Format oneof)",
			msg: &waE2E.Message{TemplateMessage: &waE2E.TemplateMessage{
				Format: &waE2E.TemplateMessage_HydratedFourRowTemplate_{
					HydratedFourRowTemplate: &waE2E.TemplateMessage_HydratedFourRowTemplate{
						Title:               &waE2E.TemplateMessage_HydratedFourRowTemplate_HydratedTitleText{HydratedTitleText: "Pedido 123"},
						HydratedContentText: proto.String("Seu pedido saiu para entrega."),
						HydratedFooterText:  proto.String("Loja Exemplo"),
						HydratedButtons: []*waE2E.HydratedTemplateButton{
							hydratedQuickReply("Recebi", "ok"),
							hydratedURLButton("Rastrear", "https://example.com/r/123"),
						},
					},
				},
			}},
			want: "Pedido 123\nSeu pedido saiu para entrega.\nLoja Exemplo\n\n1. Recebi\n2. Rastrear: https://example.com/r/123",
		},
		{
			name: "hydrated template in standalone HydratedTemplate field renders once",
			msg: &waE2E.Message{TemplateMessage: &waE2E.TemplateMessage{
				HydratedTemplate: &waE2E.TemplateMessage_HydratedFourRowTemplate{
					HydratedContentText: proto.String("Responda a pesquisa"),
					HydratedButtons:     []*waE2E.HydratedTemplateButton{hydratedQuickReply("Ótimo", "5"), hydratedQuickReply("Ruim", "1")},
				},
			}},
			want: "Responda a pesquisa\n\n1. Ótimo\n2. Ruim",
		},
		{
			name: "hydrated template in both placements renders once",
			msg: func() *waE2E.Message {
				hydrated := &waE2E.TemplateMessage_HydratedFourRowTemplate{
					HydratedContentText: proto.String("Olá"),
					HydratedButtons:     []*waE2E.HydratedTemplateButton{hydratedQuickReply("Sim", "1")},
				}
				return &waE2E.Message{TemplateMessage: &waE2E.TemplateMessage{
					Format:           &waE2E.TemplateMessage_HydratedFourRowTemplate_{HydratedFourRowTemplate: hydrated},
					HydratedTemplate: hydrated,
				}}
			}(),
			want: "Olá\n\n1. Sim",
		},
		{
			name: "notification template without buttons renders hydrated text",
			msg: &waE2E.Message{TemplateMessage: &waE2E.TemplateMessage{
				Format: &waE2E.TemplateMessage_HydratedFourRowTemplate_{
					HydratedFourRowTemplate: &waE2E.TemplateMessage_HydratedFourRowTemplate{
						HydratedContentText: proto.String("Seu código de acesso é 4821."),
						HydratedFooterText:  proto.String("Não compartilhe"),
					},
				},
			}},
			want: "Seu código de acesso é 4821.\nNão compartilhe",
		},
		{
			name: "interactive message template with native-flow buttons",
			msg: &waE2E.Message{TemplateMessage: &waE2E.TemplateMessage{
				Format: &waE2E.TemplateMessage_InteractiveMessageTemplate{
					InteractiveMessageTemplate: nativeFlowInteractive("Sua fatura está disponível",
						nativeFlowButton("quick_reply", `{"display_text":"Já paguei","id":"paid"}`),
						nativeFlowButton("cta_url", `{"display_text":"Ver fatura","url":"https://example.com/f/9","merchant_url":"https://example.com"}`),
						nativeFlowButton("voice_call", `{"display_text":"Ligar para a central"}`),
					),
				},
			}},
			want: "Sua fatura está disponível\n\n1. Já paguei\n2. Ver fatura: https://example.com/f/9\n3. Ligar para a central",
		},
		{
			name: "interactive cta_url with header title and footer",
			msg: &waE2E.Message{InteractiveMessage: func() *waE2E.InteractiveMessage {
				im := nativeFlowInteractive("Conheça nossa nova coleção",
					nativeFlowButton("cta_url", `{"display_text":"Abrir site","url":"https://example.com/colecao"}`))
				im.Header = &waE2E.InteractiveMessage_Header{Title: proto.String("Novidades"), HasMediaAttachment: proto.Bool(true)}
				im.Footer = &waE2E.InteractiveMessage_Footer{Text: proto.String("Loja Exemplo")}
				return im
			}()},
			want: "Novidades\nConheça nossa nova coleção\nLoja Exemplo\n\n1. Abrir site: https://example.com/colecao",
		},
		{
			name: "interactive galaxy_message (WhatsApp Flow)",
			msg: &waE2E.Message{InteractiveMessage: nativeFlowInteractive("Preencha o cadastro",
				nativeFlowButton("galaxy_message", `{"flow_message_version":"3","flow_token":"tok","flow_id":"123","flow_cta":"Começar","flow_action":"navigate"}`))},
			want: "Preencha o cadastro\n\n1. Começar (formulário — abrir no celular)",
		},
		{
			name: "interactive payment_info (Pix)",
			msg: &waE2E.Message{InteractiveMessage: nativeFlowInteractive("Pagamento do pedido #77",
				nativeFlowButton("payment_info", `{"currency":"BRL","total_amount":{"value":1500,"offset":100},"reference_id":"77","payment_settings":[{"type":"pix_static_code","pix_static_code":{"merchant_name":"Loja","key":"x","key_type":"EMAIL"}}]}`))},
			want: "Pagamento do pedido #77\n\n1. Pagamento Pix (abrir no celular)",
		},
		{
			name: "interactive review_and_pay without pix settings",
			msg: &waE2E.Message{InteractiveMessage: nativeFlowInteractive("Revise seu pedido",
				nativeFlowButton("review_and_pay", `{"currency":"BRL","reference_id":"88"}`))},
			want: "Revise seu pedido\n\n1. Pagamento (abrir no celular)",
		},
		{
			name: "interactive single_select renders rows as options",
			msg: &waE2E.Message{InteractiveMessage: nativeFlowInteractive("Escolha uma das opções",
				nativeFlowButton("single_select", `{"title":"Ver opções","sections":[{"title":"Assuntos","rows":[{"header":"","title":"Pedidos","description":"","id":"1"},{"title":"Financeiro","description":"Boletos e notas","id":"2"}]}]}`))},
			want: "Escolha uma das opções\n\n[Ver opções]\nAssuntos\n1. Pedidos\n2. Financeiro — Boletos e notas",
		},
		{
			name: "interactive cta_copy and cta_call",
			msg: &waE2E.Message{InteractiveMessage: nativeFlowInteractive("Use o cupom",
				nativeFlowButton("cta_copy", `{"display_text":"Copiar cupom","copy_code":"BEMVINDO10"}`),
				nativeFlowButton("cta_call", `{"display_text":"Ligar","phone_number":"+5521999999999"}`))},
			want: "Use o cupom\n\n1. Copiar cupom: BEMVINDO10\n2. Ligar: +5521999999999",
		},
		{
			name: "interactive button with invalid params falls back to name",
			msg: &waE2E.Message{InteractiveMessage: nativeFlowInteractive("Olá",
				nativeFlowButton("mystery_button", `{not json`))},
			want: "Olá\n\n1. mystery_button",
		},
		{
			name: "interactive card without buttons renders its text",
			msg: &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{
				Header: &waE2E.InteractiveMessage_Header{HasMediaAttachment: proto.Bool(true)},
				Body:   &waE2E.InteractiveMessage_Body{Text: proto.String("Aulas de natação para todas as idades")},
				InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
					NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{},
				},
			}},
			want: "Aulas de natação para todas as idades",
		},
		{
			name: "interactive carousel renders each card",
			msg: &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{
				Body: &waE2E.InteractiveMessage_Body{Text: proto.String("Confira os planos")},
				InteractiveMessage: &waE2E.InteractiveMessage_CarouselMessage_{
					CarouselMessage: &waE2E.InteractiveMessage_CarouselMessage{
						Cards: []*waE2E.InteractiveMessage{
							nativeFlowInteractive("Plano Básico", nativeFlowButton("quick_reply", `{"display_text":"Quero o básico","id":"b"}`)),
							nativeFlowInteractive("Plano Plus", nativeFlowButton("cta_url", `{"display_text":"Detalhes","url":"https://example.com/plus"}`)),
						},
						MessageVersion: proto.Int32(1),
					},
				},
			}},
			want: "Confira os planos\n\nPlano Básico\n\n1. Quero o básico\n\nPlano Plus\n\n1. Detalhes: https://example.com/plus",
		},
		{
			name: "empty menu renders nothing",
			msg:  &waE2E.Message{ButtonsMessage: &waE2E.ButtonsMessage{}},
			want: "",
		},
		{
			name: "list response uses title (DT3 pick)",
			msg: &waE2E.Message{ListResponseMessage: &waE2E.ListResponseMessage{
				Title:             proto.String("Pedidos"),
				ListType:          waE2E.ListResponseMessage_SINGLE_SELECT.Enum(),
				SingleSelectReply: &waE2E.ListResponseMessage_SingleSelectReply{SelectedRowID: proto.String("1")},
			}},
			want: "Pedidos",
		},
		{
			name: "list response falls back to description then row id",
			msg: &waE2E.Message{ListResponseMessage: &waE2E.ListResponseMessage{
				SingleSelectReply: &waE2E.ListResponseMessage_SingleSelectReply{SelectedRowID: proto.String("row-7")},
			}},
			want: "row-7",
		},
		{
			name: "list response with only description",
			msg: &waE2E.Message{ListResponseMessage: &waE2E.ListResponseMessage{
				Description:       proto.String("Boletos e notas"),
				SingleSelectReply: &waE2E.ListResponseMessage_SingleSelectReply{SelectedRowID: proto.String("2")},
			}},
			want: "Boletos e notas",
		},
		{
			name: "buttons response uses display text",
			msg: &waE2E.Message{ButtonsResponseMessage: &waE2E.ButtonsResponseMessage{
				Response:         &waE2E.ButtonsResponseMessage_SelectedDisplayText{SelectedDisplayText: "Sim"},
				SelectedButtonID: proto.String("sim"),
				Type:             waE2E.ButtonsResponseMessage_DISPLAY_TEXT.Enum(),
			}},
			want: "Sim",
		},
		{
			name: "buttons response falls back to button id",
			msg: &waE2E.Message{ButtonsResponseMessage: &waE2E.ButtonsResponseMessage{
				SelectedButtonID: proto.String("nao"),
			}},
			want: "nao",
		},
		{
			name: "template button reply uses display text",
			msg: &waE2E.Message{TemplateButtonReplyMessage: &waE2E.TemplateButtonReplyMessage{
				SelectedID:          proto.String("ok"),
				SelectedDisplayText: proto.String("Recebi"),
				SelectedIndex:       proto.Uint32(0),
			}},
			want: "Recebi",
		},
		{
			name: "template button reply falls back to id",
			msg: &waE2E.Message{TemplateButtonReplyMessage: &waE2E.TemplateButtonReplyMessage{
				SelectedID: proto.String("ok"),
			}},
			want: "ok",
		},
		{
			name: "interactive response uses body text",
			msg: &waE2E.Message{InteractiveResponseMessage: &waE2E.InteractiveResponseMessage{
				Body: &waE2E.InteractiveResponseMessage_Body{Text: proto.String("Já paguei"), Format: waE2E.InteractiveResponseMessage_Body_DEFAULT.Enum()},
				InteractiveResponseMessage: &waE2E.InteractiveResponseMessage_NativeFlowResponseMessage_{
					NativeFlowResponseMessage: &waE2E.InteractiveResponseMessage_NativeFlowResponseMessage{
						Name:       proto.String("quick_reply"),
						ParamsJSON: proto.String(`{"id":"paid"}`),
						Version:    proto.Int32(1),
					},
				},
			}},
			want: "Já paguei",
		},
		{
			name: "interactive response falls back to params id then name",
			msg: &waE2E.Message{InteractiveResponseMessage: &waE2E.InteractiveResponseMessage{
				InteractiveResponseMessage: &waE2E.InteractiveResponseMessage_NativeFlowResponseMessage_{
					NativeFlowResponseMessage: &waE2E.InteractiveResponseMessage_NativeFlowResponseMessage{
						Name:       proto.String("single_select"),
						ParamsJSON: proto.String(`{"id":"2"}`),
					},
				},
			}},
			want: "2",
		},
		{
			name: "interactive response with unparseable params uses name",
			msg: &waE2E.Message{InteractiveResponseMessage: &waE2E.InteractiveResponseMessage{
				InteractiveResponseMessage: &waE2E.InteractiveResponseMessage_NativeFlowResponseMessage_{
					NativeFlowResponseMessage: &waE2E.InteractiveResponseMessage_NativeFlowResponseMessage{
						Name:       proto.String("galaxy_message"),
						ParamsJSON: proto.String(`{"flow_token":`),
					},
				},
			}},
			want: "galaxy_message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractInteractiveText(tt.msg))
		})
	}
}

// TestExtractMessageTextFromProtoRendersMenus covers the chat-storage and
// history-sync path, which reads text through ExtractMessageTextFromProto.
func TestExtractMessageTextFromProtoRendersMenus(t *testing.T) {
	menu := ExtractMessageTextFromProto(&waE2E.Message{ListMessage: dt3ListMessage()})
	assert.Contains(t, menu, "Escolha uma das opções")
	assert.Contains(t, menu, "1. Pedidos")

	pick := ExtractMessageTextFromProto(&waE2E.Message{ListResponseMessage: &waE2E.ListResponseMessage{
		Title:             proto.String("Pedidos"),
		SingleSelectReply: &waE2E.ListResponseMessage_SingleSelectReply{SelectedRowID: proto.String("1")},
	}})
	assert.Equal(t, "Pedidos", pick)

	template := ExtractMessageTextFromProto(&waE2E.Message{TemplateMessage: &waE2E.TemplateMessage{
		Format: &waE2E.TemplateMessage_HydratedFourRowTemplate_{
			HydratedFourRowTemplate: &waE2E.TemplateMessage_HydratedFourRowTemplate{
				HydratedContentText: proto.String("Seu código é 4821."),
			},
		},
	}})
	assert.Equal(t, "Seu código é 4821.", template)

	interactivePick := ExtractMessageTextFromProto(&waE2E.Message{InteractiveResponseMessage: &waE2E.InteractiveResponseMessage{
		Body: &waE2E.InteractiveResponseMessage_Body{Text: proto.String("Já paguei")},
	}})
	assert.Equal(t, "Já paguei", interactivePick)

	assert.Equal(t, "oi", ExtractMessageTextFromProto(&waE2E.Message{Conversation: proto.String("oi")}))
}
