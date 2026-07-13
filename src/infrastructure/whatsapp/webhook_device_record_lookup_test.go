package whatsapp

import (
	"testing"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
)

// jidKeyedStubStorage models the auto-created slot shape: the row is keyed BY the JID
// (device_id = "<jid>") and its `jid` column is empty, because handleConnectionEvents skips
// persisting jid for ids containing "@" (it would recreate deleted duplicates). A lookup by
// the jid COLUMN therefore finds nothing.
type jidKeyedStubStorage struct {
	domainChatStorage.IChatStorageRepository
	byID map[string]*domainChatStorage.DeviceRecord
}

func (s *jidKeyedStubStorage) GetDeviceRecordByJID(string) (*domainChatStorage.DeviceRecord, error) {
	return nil, nil // no row carries this value in its jid column
}

func (s *jidKeyedStubStorage) GetDeviceRecord(deviceID string) (*domainChatStorage.DeviceRecord, error) {
	return s.byID[deviceID], nil
}

// A webhook configured on a JID-keyed slot before its first pairing must still be found:
// otherwise the device silently falls through to the global (or no-op) path and its
// consumer never receives anything.
func TestGetWebhookConfigForDevice_FallsBackToTheJIDKeyedSlotRecord(t *testing.T) {
	resetWebhookCaches()
	defer resetWebhookCaches()

	const jid = "628123450050@s.whatsapp.net"
	url := "https://jid-keyed.example/hook"

	storage := &jidKeyedStubStorage{byID: map[string]*domainChatStorage.DeviceRecord{
		jid: {DeviceID: jid, JID: "", WebhookURL: &url, WebhookSecret: "s3cret"},
	}}

	origManager := deviceManager
	deviceManager = NewDeviceManager(nil, nil, storage)
	defer func() { deviceManager = origManager }()

	cfg, err := getWebhookConfigForDevice(jid)
	if err != nil {
		t.Fatalf("getWebhookConfigForDevice: %v", err)
	}
	if cfg == nil || cfg.WebhookURL == nil || *cfg.WebhookURL != url {
		t.Fatalf("expected the JID-keyed slot's webhook %q to be resolved, got %+v", url, cfg)
	}
	if cfg.WebhookSecret != "s3cret" {
		t.Fatalf("expected the device secret to be carried through, got %q", cfg.WebhookSecret)
	}
}
