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

// Scenario: device A owns a per-device webhook, device B owns nothing, and there is no
// global webhook or Chatwoot. The fleet-wide gate answers "a consumer exists" for BOTH
// (some device has one), so B's media messages would be downloaded to disk during
// payload construction and then dropped for want of a destination. The device-scoped
// gate must admit A and hold B.
func TestHasWebhookConsumerForDevice_ScopesToTheDeviceNotTheFleet(t *testing.T) {
	resetWebhookCaches()
	defer resetWebhookCaches()

	origWebhook, origChatwoot := config.WhatsappWebhook, config.ChatwootEnabled
	config.WhatsappWebhook = nil
	config.ChatwootEnabled = false
	defer func() { config.WhatsappWebhook, config.ChatwootEnabled = origWebhook, origChatwoot }()

	const withHook = "628123450010@s.whatsapp.net"
	const withoutHook = "628123450011@s.whatsapp.net"
	url := "https://tenant-a.example/hook"

	origStorage := webhookStorageForTest
	webhookStorageForTest = func(jid string) (*domainChatStorage.DeviceRecord, error) {
		if jid == withHook {
			return &domainChatStorage.DeviceRecord{DeviceID: jid, JID: jid, WebhookURL: &url}, nil
		}
		return &domainChatStorage.DeviceRecord{DeviceID: jid, JID: jid}, nil
	}
	defer func() { webhookStorageForTest = origStorage }()

	if !hasWebhookConsumerForDevice(withHook) {
		t.Fatal("expected the device owning the per-device webhook to be a consumer")
	}
	if hasWebhookConsumerForDevice(withoutHook) {
		t.Fatal("expected a device with no destination of its own to be gated out, " +
			"even though another device owns a webhook")
	}
}

// The device-scoped gate keeps the global legs: a global webhook or Chatwoot is a
// destination for every device, whatever its own config says.
func TestHasWebhookConsumerForDevice_GlobalLegsAdmitEveryDevice(t *testing.T) {
	resetWebhookCaches()
	defer resetWebhookCaches()

	origWebhook, origChatwoot := config.WhatsappWebhook, config.ChatwootEnabled
	defer func() { config.WhatsappWebhook, config.ChatwootEnabled = origWebhook, origChatwoot }()

	const bare = "628123450012@s.whatsapp.net"
	origStorage := webhookStorageForTest
	webhookStorageForTest = func(jid string) (*domainChatStorage.DeviceRecord, error) {
		return &domainChatStorage.DeviceRecord{DeviceID: jid, JID: jid}, nil
	}
	defer func() { webhookStorageForTest = origStorage }()

	config.WhatsappWebhook = []string{"https://example.test/hook"}
	config.ChatwootEnabled = false
	if !hasWebhookConsumerForDevice(bare) {
		t.Fatal("expected the global webhook to be a consumer for a device with no own config")
	}

	config.WhatsappWebhook = nil
	config.ChatwootEnabled = true
	if !hasWebhookConsumerForDevice(bare) {
		t.Fatal("expected Chatwoot to be a consumer for a device with no own config")
	}
}

// A per-device config lookup failure must fail OPEN: dropping the event here would
// silently lose messages for a device that may well have a webhook, which is worse than
// building a payload that the delivery path then discards.
func TestHasWebhookConsumerForDevice_LookupErrorFailsOpen(t *testing.T) {
	resetWebhookCaches()
	defer resetWebhookCaches()

	origWebhook, origChatwoot := config.WhatsappWebhook, config.ChatwootEnabled
	config.WhatsappWebhook = nil
	config.ChatwootEnabled = false
	defer func() { config.WhatsappWebhook, config.ChatwootEnabled = origWebhook, origChatwoot }()

	origStorage := webhookStorageForTest
	webhookStorageForTest = func(string) (*domainChatStorage.DeviceRecord, error) {
		return nil, errors.New("database is locked")
	}
	defer func() { webhookStorageForTest = origStorage }()

	if !hasWebhookConsumerForDevice("628123450013@s.whatsapp.net") {
		t.Fatal("expected a lookup error to fail open rather than drop the event")
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

// A device webhook config owns its own event filter. An EMPTY device event list means
// "all events" (what openapi.yaml documents, and the same empty = no filter rule the
// global list follows) — it must NOT inherit the global whitelist, which would silently
// drop event types from a device webhook that asked for no filter at all.
func TestIsEventWhitelistedForDevice_EmptyDeviceEventListMeansAllEvents(t *testing.T) {
	origEvents := config.WhatsappWebhookEvents
	defer func() { config.WhatsappWebhookEvents = origEvents }()

	// Global deployment restricts events to "message" only.
	config.WhatsappWebhookEvents = []string{"message"}

	url := "https://tenant-a.example/hook"
	deviceNoFilter := &domainChatStorage.DeviceWebhookConfig{WebhookURL: &url} // events unset

	for _, event := range []string{"message", "call.offer", "group.participants", "receipt"} {
		if !isEventWhitelistedForDevice(event, deviceNoFilter) {
			t.Fatalf("device webhook with an empty event list must receive %q (empty = all events)", event)
		}
	}

	// An explicit device list still filters, and is not widened by the global one.
	deviceFiltered := &domainChatStorage.DeviceWebhookConfig{WebhookURL: &url, WebhookEvents: "call.offer"}
	if !isEventWhitelistedForDevice("call.offer", deviceFiltered) {
		t.Fatal("expected an explicitly listed event to pass")
	}
	if isEventWhitelistedForDevice("message", deviceFiltered) {
		t.Fatal("expected an event absent from the device list to be filtered out, despite the global list allowing it")
	}

	// A device with NO config of its own still falls back to the global whitelist.
	if isEventWhitelistedForDevice("call.offer", nil) {
		t.Fatal("expected the global whitelist to apply when the device has no webhook config")
	}
	if !isEventWhitelistedForDevice("message", nil) {
		t.Fatal("expected the globally whitelisted event to pass for a device with no config")
	}
}
