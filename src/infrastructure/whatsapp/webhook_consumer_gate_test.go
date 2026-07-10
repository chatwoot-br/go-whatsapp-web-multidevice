package whatsapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// resetWebhookCaches clears the device webhook caches between tests.
func resetWebhookCaches() {
	deviceWebhookConfigCache.Clear()
	anyDeviceWebhookCache.Store(&anyDeviceWebhookEntry{})
}

// Scenario: a transient storage error during the per-device webhook config lookup
// must NOT divert a device with a known webhook override to the global URL — the
// last-known config (even past its TTL) is served instead.
func TestGetWebhookConfigForDevice_LookupError_ServesLastKnownConfig(t *testing.T) {
	resetWebhookCaches()
	defer resetWebhookCaches()

	deviceJID := "628123450001@s.whatsapp.net"
	url := "https://tenant-a.example/hook"
	known := &domainChatStorage.DeviceWebhookConfig{WebhookURL: &url, WebhookSecret: "s3cret"}
	// Expired entry: only the error-grace path may serve it.
	deviceWebhookConfigCache.Store(deviceJID, deviceWebhookConfigCacheEntry{
		config:    known,
		expiresAt: time.Now().Add(-time.Minute),
	})

	origStorage := webhookStorageForTest
	webhookStorageForTest = func(string) (*domainChatStorage.DeviceRecord, error) {
		return nil, errors.New("database is locked")
	}
	defer func() { webhookStorageForTest = origStorage }()

	got, err := getWebhookConfigForDevice(deviceJID)
	if err != nil {
		t.Fatalf("expected last-known config served on lookup error, got error: %v", err)
	}
	if got == nil || got.WebhookURL == nil || *got.WebhookURL != url {
		t.Fatalf("expected last-known device config %q, got %+v", url, got)
	}
}

// Scenario: with no cached knowledge at all, a lookup error keeps upstream's
// fall-back-to-global contract (the caller logs and uses the global config).
func TestGetWebhookConfigForDevice_LookupError_NoCache_ReturnsError(t *testing.T) {
	resetWebhookCaches()
	defer resetWebhookCaches()

	origStorage := webhookStorageForTest
	webhookStorageForTest = func(string) (*domainChatStorage.DeviceRecord, error) {
		return nil, errors.New("database is locked")
	}
	defer func() { webhookStorageForTest = origStorage }()

	if _, err := getWebhookConfigForDevice("628123450002@s.whatsapp.net"); err == nil {
		t.Fatal("expected the lookup error to propagate when nothing is cached")
	}
}

// Scenario: no consumers at all (no global webhook, Chatwoot off, no device webhook
// rows) — handleWebhookForward must return before spawning the forward goroutine,
// because payload construction downloads media to disk. The old pre-#671 zero-cost
// fast path is restored by the hasAnyWebhookConsumer gate.
func TestHandleWebhookForward_NoConsumers_SkipsForward(t *testing.T) {
	resetWebhookCaches()
	defer resetWebhookCaches()

	log = waLog.Noop
	origWebhook, origChatwoot := config.WhatsappWebhook, config.ChatwootEnabled
	config.WhatsappWebhook = nil
	config.ChatwootEnabled = false
	defer func() { config.WhatsappWebhook, config.ChatwootEnabled = origWebhook, origChatwoot }()

	// Cache "no device webhooks exist" so the gate resolves without a DeviceManager.
	anyDeviceWebhookCache.Store(&anyDeviceWebhookEntry{exists: false, expiresAt: time.Now().Add(time.Minute)})

	delivered := make(chan map[string]any, 1)
	origSubmit := submitWebhookFn
	submitWebhookFn = func(_ context.Context, payload map[string]any, _ string, _ *domainChatStorage.DeviceWebhookConfig) error {
		delivered <- payload
		return nil
	}
	defer func() { submitWebhookFn = origSubmit }()

	dmChat := types.NewJID("628123450003", types.DefaultUserServer)
	handleWebhookForward(context.Background(), textEventForTest("no-consumer-1", dmChat), nil, nil)

	select {
	case payload := <-delivered:
		t.Fatalf("expected no forwarding with zero consumers, got delivery: %v", payload)
	case <-time.After(300 * time.Millisecond):
		// No goroutine spawned — the gate held.
	}
}

// hasAnyWebhookConsumer leg-by-leg: global webhook, Chatwoot, cached device webhook.
func TestHasAnyWebhookConsumer(t *testing.T) {
	resetWebhookCaches()
	defer resetWebhookCaches()

	origWebhook, origChatwoot := config.WhatsappWebhook, config.ChatwootEnabled
	defer func() { config.WhatsappWebhook, config.ChatwootEnabled = origWebhook, origChatwoot }()

	config.WhatsappWebhook = nil
	config.ChatwootEnabled = false
	anyDeviceWebhookCache.Store(&anyDeviceWebhookEntry{exists: false, expiresAt: time.Now().Add(time.Minute)})
	if hasAnyWebhookConsumer() {
		t.Fatal("expected false with no consumers")
	}

	config.WhatsappWebhook = []string{"https://example.test/hook"}
	if !hasAnyWebhookConsumer() {
		t.Fatal("expected true with a global webhook")
	}

	config.WhatsappWebhook = nil
	config.ChatwootEnabled = true
	if !hasAnyWebhookConsumer() {
		t.Fatal("expected true with Chatwoot enabled")
	}

	config.ChatwootEnabled = false
	anyDeviceWebhookCache.Store(&anyDeviceWebhookEntry{exists: true, expiresAt: time.Now().Add(time.Minute)})
	if !hasAnyWebhookConsumer() {
		t.Fatal("expected true with a cached device webhook")
	}
}
