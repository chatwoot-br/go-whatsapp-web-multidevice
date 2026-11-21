package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// maxCarouselDepth bounds recursion through carousel cards, which are
// themselves InteractiveMessages and could in theory nest further.
const maxCarouselDepth = 2

// ExtractInteractiveText renders business option menus (list, buttons,
// template and interactive messages) and the options picked from them (list,
// buttons, template-button and interactive responses) as readable plain text.
//
// Menus render as header/title, body, footer and then the options as a
// numbered list, numbered in one sequence across list sections. Option titles
// are kept verbatim so agents can type them back. Picks render as the chosen
// option's title, falling back to its description and then its id.
//
// It returns "" when msg is not one of these types or carries nothing
// renderable, so callers can keep skipping empty messages.
func ExtractInteractiveText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}

	switch {
	case msg.GetListMessage() != nil:
		return renderListMessage(msg.GetListMessage())
	case msg.GetButtonsMessage() != nil:
		return renderButtonsMessage(msg.GetButtonsMessage())
	case msg.GetTemplateMessage() != nil:
		return renderTemplateMessage(msg.GetTemplateMessage())
	case msg.GetInteractiveMessage() != nil:
		return renderInteractiveMessage(msg.GetInteractiveMessage(), 0)
	case msg.GetListResponseMessage() != nil:
		r := msg.GetListResponseMessage()
		return firstNonEmpty(r.GetTitle(), r.GetDescription(), r.GetSingleSelectReply().GetSelectedRowID())
	case msg.GetButtonsResponseMessage() != nil:
		r := msg.GetButtonsResponseMessage()
		return firstNonEmpty(r.GetSelectedDisplayText(), r.GetSelectedButtonID())
	case msg.GetTemplateButtonReplyMessage() != nil:
		r := msg.GetTemplateButtonReplyMessage()
		return firstNonEmpty(r.GetSelectedDisplayText(), r.GetSelectedID())
	case msg.GetInteractiveResponseMessage() != nil:
		return renderInteractiveResponse(msg.GetInteractiveResponseMessage())
	}

	return ""
}

// menuText accumulates the text blocks and numbered options of one menu.
type menuText struct {
	head    []string
	options []string
	next    int
}

func (m *menuText) addHead(parts ...string) {
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			m.head = append(m.head, p)
		}
	}
}

// addLabel adds an unnumbered line among the options (section titles, the
// button that opens a list). A label that follows numbered options gets a
// blank line first: in markdown (Chatwoot renders it) a plain line right
// after a list item is folded into that item.
func (m *menuText) addLabel(label string) {
	if label = strings.TrimSpace(label); label == "" {
		return
	}
	if m.next > 0 {
		m.options = append(m.options, "")
	}
	m.options = append(m.options, label)
}

func (m *menuText) addOption(option string) {
	if option = strings.TrimSpace(option); option == "" {
		return
	}
	m.next++
	m.options = append(m.options, fmt.Sprintf("%d. %s", m.next, option))
}

func (m *menuText) String() string {
	blocks := make([]string, 0, 2)
	if len(m.head) > 0 {
		blocks = append(blocks, strings.Join(m.head, "\n"))
	}
	if m.next > 0 {
		blocks = append(blocks, strings.Join(m.options, "\n"))
	}
	return strings.Join(blocks, "\n\n")
}

type menuRow struct {
	title       string
	description string
}

func (m *menuText) addRows(rows []menuRow) {
	for _, row := range rows {
		title := strings.TrimSpace(row.title)
		desc := strings.TrimSpace(row.description)
		switch {
		case title != "" && desc != "":
			m.addOption(title + " — " + desc)
		case title != "":
			m.addOption(title)
		default:
			m.addOption(desc)
		}
	}
}

func renderListMessage(list *waE2E.ListMessage) string {
	var m menuText
	m.addHead(list.GetTitle(), list.GetDescription(), list.GetFooterText())
	m.addLabel(bracketLabel(list.GetButtonText()))
	for _, section := range list.GetSections() {
		m.addLabel(section.GetTitle())
		rows := make([]menuRow, 0, len(section.GetRows()))
		for _, row := range section.GetRows() {
			rows = append(rows, menuRow{title: row.GetTitle(), description: row.GetDescription()})
		}
		m.addRows(rows)
	}
	return m.String()
}

func renderButtonsMessage(buttons *waE2E.ButtonsMessage) string {
	var m menuText
	m.addHead(buttons.GetText(), buttons.GetContentText(), buttons.GetFooterText())
	for _, button := range buttons.GetButtons() {
		label := button.GetButtonText().GetDisplayText()
		if label == "" && button.GetNativeFlowInfo() != nil {
			info := button.GetNativeFlowInfo()
			addNativeFlowButton(&m, info.GetName(), info.GetParamsJSON())
			continue
		}
		m.addOption(label)
	}
	return m.String()
}

func renderTemplateMessage(tmpl *waE2E.TemplateMessage) string {
	if interactive := tmpl.GetInteractiveMessageTemplate(); interactive != nil {
		return renderInteractiveMessage(interactive, 0)
	}
	// The hydrated form can sit in the Format oneof or in the standalone
	// HydratedTemplate field; prefer the oneof and render it only once.
	hydrated := tmpl.GetHydratedFourRowTemplate()
	if hydrated == nil {
		hydrated = tmpl.GetHydratedTemplate()
	}
	if hydrated == nil {
		return ""
	}

	var m menuText
	m.addHead(hydrated.GetHydratedTitleText(), hydrated.GetHydratedContentText(), hydrated.GetHydratedFooterText())
	for _, button := range hydrated.GetHydratedButtons() {
		switch {
		case button.GetQuickReplyButton() != nil:
			b := button.GetQuickReplyButton()
			m.addOption(firstNonEmpty(b.GetDisplayText(), b.GetID()))
		case button.GetUrlButton() != nil:
			b := button.GetUrlButton()
			m.addOption(labelWithValue(b.GetDisplayText(), b.GetURL()))
		case button.GetCallButton() != nil:
			b := button.GetCallButton()
			m.addOption(labelWithValue(b.GetDisplayText(), b.GetPhoneNumber()))
		}
	}
	return m.String()
}

func renderInteractiveMessage(im *waE2E.InteractiveMessage, depth int) string {
	var m menuText
	header := im.GetHeader()
	m.addHead(header.GetTitle(), header.GetSubtitle(), im.GetBody().GetText(), im.GetFooter().GetText())

	if nativeFlow := im.GetNativeFlowMessage(); nativeFlow != nil {
		for _, button := range nativeFlow.GetButtons() {
			addNativeFlowButton(&m, button.GetName(), button.GetButtonParamsJSON())
		}
	}
	text := m.String()

	carousel := im.GetCarouselMessage()
	if carousel == nil || depth >= maxCarouselDepth {
		return text
	}
	blocks := make([]string, 0, len(carousel.GetCards())+1)
	if text != "" {
		blocks = append(blocks, text)
	}
	for _, card := range carousel.GetCards() {
		if card == nil {
			continue
		}
		if cardText := renderInteractiveMessage(card, depth+1); cardText != "" {
			blocks = append(blocks, cardText)
		}
	}
	return strings.Join(blocks, "\n\n")
}

// nativeFlowParams holds the buttonParamsJSON / paramsJSON keys GoWA reads.
// Parsing is lenient: unknown keys are ignored and invalid JSON yields the
// zero value, so a malformed button still renders by its name.
type nativeFlowParams struct {
	DisplayText string `json:"display_text"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ID          string `json:"id"`
	URL         string `json:"url"`
	PhoneNumber string `json:"phone_number"`
	CopyCode    string `json:"copy_code"`
	FlowCTA     string `json:"flow_cta"`
	Sections    []struct {
		Title string `json:"title"`
		Rows  []struct {
			Header      string `json:"header"`
			Title       string `json:"title"`
			Description string `json:"description"`
			ID          string `json:"id"`
		} `json:"rows"`
	} `json:"sections"`
	PaymentSettings []struct {
		Type string `json:"type"`
	} `json:"payment_settings"`
}

func parseNativeFlowParams(raw string) nativeFlowParams {
	var p nativeFlowParams
	if raw = strings.TrimSpace(raw); raw != "" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	return p
}

// addNativeFlowButton renders one native-flow button. Buttons that only work
// on the phone (flows, payments) are labeled as such so agents know why they
// cannot be answered by typing.
func addNativeFlowButton(m *menuText, name, paramsJSON string) {
	p := parseNativeFlowParams(paramsJSON)
	label := firstNonEmpty(p.DisplayText, p.Title)

	switch name {
	case "cta_url":
		m.addOption(labelWithValue(firstNonEmpty(label, p.URL), p.URL))
	case "cta_call":
		m.addOption(labelWithValue(firstNonEmpty(label, p.PhoneNumber), p.PhoneNumber))
	case "cta_copy":
		m.addOption(labelWithValue(firstNonEmpty(label, p.CopyCode), p.CopyCode))
	case "single_select":
		m.addLabel(bracketLabel(label))
		for _, section := range p.Sections {
			m.addLabel(section.Title)
			rows := make([]menuRow, 0, len(section.Rows))
			for _, row := range section.Rows {
				rows = append(rows, menuRow{
					title:       firstNonEmpty(row.Title, row.Header, row.ID),
					description: row.Description,
				})
			}
			m.addRows(rows)
		}
	case "galaxy_message":
		m.addOption(firstNonEmpty(p.FlowCTA, label, "Formulário") + " (formulário — abrir no celular)")
	case "payment_info", "review_and_pay", "review_order":
		m.addOption(paymentLabel(p) + " (abrir no celular)")
	case "voice_call":
		m.addOption(firstNonEmpty(label, "Ligar"))
	default:
		m.addOption(firstNonEmpty(label, p.ID, name))
	}
}

func paymentLabel(p nativeFlowParams) string {
	for _, setting := range p.PaymentSettings {
		if strings.Contains(strings.ToLower(setting.Type), "pix") {
			return "Pagamento Pix"
		}
	}
	return "Pagamento"
}

func renderInteractiveResponse(r *waE2E.InteractiveResponseMessage) string {
	if text := strings.TrimSpace(r.GetBody().GetText()); text != "" {
		return text
	}
	flow := r.GetNativeFlowResponseMessage()
	if flow == nil {
		return ""
	}
	p := parseNativeFlowParams(flow.GetParamsJSON())
	return firstNonEmpty(p.Title, p.DisplayText, p.Description, p.ID, flow.GetName())
}

func bracketLabel(label string) string {
	if label = strings.TrimSpace(label); label == "" {
		return ""
	}
	return "[" + label + "]"
}

func labelWithValue(label, value string) string {
	label = strings.TrimSpace(label)
	value = strings.TrimSpace(value)
	if label == "" || label == value {
		return value
	}
	if value == "" {
		return label
	}
	return label + ": " + value
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
