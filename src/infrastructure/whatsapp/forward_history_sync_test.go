package whatsapp

import (
	"context"
	"testing"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
)

// TestForwardHistorySyncCompleteToWebhook_PayloadShape locks the fork-specific
// {event, device_id, payload:{sync_type, timestamp}} payload contract emitted
// to webhooks. Uses the established submitWebhookFn var-swap pattern.
//
// The whatsmeow.Client argument is nil — exercises the "no device" branch
// where device_id falls through to "" but the payload shape must still be
// emitted verbatim. The shape itself is byte-locked by docs/webhook-payload.md.
func TestForwardHistorySyncCompleteToWebhook_PayloadShape(t *testing.T) {
	ctx := context.Background()

	origWebhooks := config.WhatsappWebhook
	config.WhatsappWebhook = []string{"https://example.test/hook"}
	defer func() { config.WhatsappWebhook = origWebhooks }()

	origEvents := config.WhatsappWebhookEvents
	config.WhatsappWebhookEvents = nil // allow all events
	defer func() { config.WhatsappWebhookEvents = origEvents }()

	var captured map[string]any
	var capturedURL string
	origSubmit := submitWebhookFn
	submitWebhookFn = func(_ context.Context, payload map[string]any, url string) error {
		captured = payload
		capturedURL = url
		return nil
	}
	defer func() { submitWebhookFn = origSubmit }()

	forwardHistorySyncCompleteToWebhook(ctx, nil, "ON_DEMAND")

	if captured == nil {
		t.Fatal("submitWebhookFn was not invoked")
	}
	if capturedURL != "https://example.test/hook" {
		t.Errorf("URL = %q, want %q", capturedURL, "https://example.test/hook")
	}

	if got := captured["event"]; got != "history_sync_complete" {
		t.Errorf("event = %v, want history_sync_complete", got)
	}
	if _, ok := captured["device_id"]; !ok {
		t.Error("missing device_id key")
	}
	inner, ok := captured["payload"].(map[string]any)
	if !ok {
		t.Fatalf("payload = %T, want map[string]any", captured["payload"])
	}
	if got := inner["sync_type"]; got != "ON_DEMAND" {
		t.Errorf("sync_type = %v, want ON_DEMAND", got)
	}
	tsStr, ok := inner["timestamp"].(string)
	if !ok {
		t.Fatalf("timestamp = %T, want string", inner["timestamp"])
	}
	if _, err := time.Parse(time.RFC3339, tsStr); err != nil {
		t.Errorf("timestamp not RFC3339: %v (%q)", err, tsStr)
	}
}

// TestForwardHistorySyncCompleteToWebhook_RespectsWhitelist exercises the
// event-name whitelist gate on the fork's new event. If `history_sync_complete`
// is missing from WHATSAPP_WEBHOOK_EVENTS, the dispatch is suppressed.
func TestForwardHistorySyncCompleteToWebhook_RespectsWhitelist(t *testing.T) {
	ctx := context.Background()

	origWebhooks := config.WhatsappWebhook
	origEvents := config.WhatsappWebhookEvents
	config.WhatsappWebhook = []string{"https://example.test/hook"}
	config.WhatsappWebhookEvents = []string{"message"} // history_sync_complete NOT whitelisted
	defer func() {
		config.WhatsappWebhook = origWebhooks
		config.WhatsappWebhookEvents = origEvents
	}()

	called := false
	origSubmit := submitWebhookFn
	submitWebhookFn = func(context.Context, map[string]any, string) error {
		called = true
		return nil
	}
	defer func() { submitWebhookFn = origSubmit }()

	forwardHistorySyncCompleteToWebhook(ctx, nil, "FULL")

	if called {
		t.Fatal("history_sync_complete should be suppressed when not in whitelist")
	}
}

// TestForwardHistorySyncCompleteToWebhook_AllowedWhenWhitelistEmpty is the
// inverse: empty whitelist means all events fire. Pairs with the test above.
func TestForwardHistorySyncCompleteToWebhook_AllowedWhenWhitelistEmpty(t *testing.T) {
	ctx := context.Background()

	origWebhooks := config.WhatsappWebhook
	origEvents := config.WhatsappWebhookEvents
	config.WhatsappWebhook = []string{"https://example.test/hook"}
	config.WhatsappWebhookEvents = nil
	defer func() {
		config.WhatsappWebhook = origWebhooks
		config.WhatsappWebhookEvents = origEvents
	}()

	called := false
	origSubmit := submitWebhookFn
	submitWebhookFn = func(context.Context, map[string]any, string) error {
		called = true
		return nil
	}
	defer func() { submitWebhookFn = origSubmit }()

	forwardHistorySyncCompleteToWebhook(ctx, nil, "RECENT")

	if !called {
		t.Fatal("expected dispatch when whitelist is empty")
	}
}
