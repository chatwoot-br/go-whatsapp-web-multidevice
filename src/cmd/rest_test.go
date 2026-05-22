package cmd

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// TestRestFiberConfigDecodesEncodedChatJID guards the UnescapePath fix.
//
// Chatwoot's history-sync client percent-encodes the chat JID, so
// /chat/:chat_jid/messages is hit as ".../5521995539939%40s.whatsapp.net/messages".
// gowa's chat-storage lookup is an exact string match — if fiber leaves "%40"
// literal, every lookup misses and the handler panics with "chat not found".
// This exercises the real production config so dropping UnescapePath fails here.
func TestRestFiberConfigDecodesEncodedChatJID(t *testing.T) {
	cfg := restFiberConfig(nil)
	assert.True(t, cfg.UnescapePath, "UnescapePath must stay enabled so :chat_jid decodes")

	app := fiber.New(cfg)
	app.Get("/chat/:chat_jid/messages", func(c *fiber.Ctx) error {
		return c.SendString(c.Params("chat_jid"))
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/chat/5521995539939%40s.whatsapp.net/messages", nil))
	assert.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "5521995539939@s.whatsapp.net", string(body),
		"path param must be percent-decoded before reaching the handler")
}
