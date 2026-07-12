package usecase

import (
	"context"
	"errors"
	"testing"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	domainDevice "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/device"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
)

// deviceStubStorage is a minimal IChatStorageRepository covering only the device
// registry calls the create/rollback path makes. webhookErr forces the webhook
// persistence to fail so the rollback is observable.
type deviceStubStorage struct {
	domainChatStorage.IChatStorageRepository
	webhookErr     error
	savedRecords   []string
	deletedRecords []string
}

func (s *deviceStubStorage) SaveDeviceRecord(rec *domainChatStorage.DeviceRecord) error {
	s.savedRecords = append(s.savedRecords, rec.DeviceID)
	return nil
}

func (s *deviceStubStorage) DeleteDeviceRecord(deviceID string) error {
	s.deletedRecords = append(s.deletedRecords, deviceID)
	return nil
}

func (s *deviceStubStorage) SetDeviceWebhookConfig(_ string, _ *domainChatStorage.DeviceWebhookConfig) error {
	return s.webhookErr
}

// Scenario: POST /devices carries webhook fields and the webhook write fails after the
// slot is already registered. Returning the error while keeping the slot makes the
// create unretryable — the same device_id then fails with "device already exists" and
// the caller cannot complete it without manual cleanup. The slot must be rolled back,
// leaving the id free for an immediate retry.
func TestAddDevice_RollsBackSlotWhenWebhookPersistenceFails(t *testing.T) {
	ctx := context.Background()
	storage := &deviceStubStorage{webhookErr: errors.New("database is locked")}
	manager := whatsapp.NewDeviceManager(nil, nil, storage)
	svc := &serviceDevice{manager: manager}

	url := "https://tenant-a.example/hook"
	webhook := &domainChatStorage.DeviceWebhookConfig{WebhookURL: &url}

	if _, err := svc.AddDevice(ctx, "dev-rollback", webhook); err == nil {
		t.Fatal("expected AddDevice to surface the webhook persistence failure")
	}

	if _, ok := manager.GetDevice("dev-rollback"); ok {
		t.Fatal("expected the slot to be rolled back after the webhook write failed")
	}

	// The id must be free: the caller retries the exact same request.
	storage.webhookErr = nil
	if _, err := svc.AddDevice(ctx, "dev-rollback", webhook); err != nil {
		t.Fatalf("expected the retry with the same device_id to succeed, got: %v", err)
	}
	if _, ok := manager.GetDevice("dev-rollback"); !ok {
		t.Fatal("expected the retry to register the slot")
	}
}

// Scenario: keep-slot logout detaches the client (ResetClient sets it to nil), so the
// logged-out slot reaches ReconnectDevice as a NIL client, never as a nil Store.ID.
// Answering with a generic "client not initialized" 500 hides the one thing the caller
// can act on; it must get the typed re-pair error instead.
func TestReconnectDevice_LoggedOutSlotReturnsRePairError(t *testing.T) {
	ctx := context.Background()
	manager := whatsapp.NewDeviceManager(nil, nil, &deviceStubStorage{})
	svc := &serviceDevice{manager: manager}

	// CreateDevice registers a slot with no client — the same shape a keep-slot
	// logout leaves behind.
	if _, err := manager.CreateDevice(ctx, "dev-logged-out"); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	err := svc.ReconnectDevice(ctx, "dev-logged-out")
	if !errors.Is(err, pkgError.ErrSessionDeleted) {
		t.Fatalf("expected ErrSessionDeleted for a slot with no client, got: %v", err)
	}
}

func TestDeviceServiceInterface(t *testing.T) {
	var _ domainDevice.IDeviceUsecase = (*serviceDevice)(nil)
}

func TestSetDeviceWebhook_InvalidManager(t *testing.T) {
	svc := &serviceDevice{manager: nil}
	err := svc.SetDeviceWebhook(context.Background(), "test-device", "https://example.com/webhook")
	if err == nil {
		t.Fatal("expected error when manager is nil")
	}
	if err.Error() != "device manager not initialized" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDeviceWebhook_InvalidManager(t *testing.T) {
	svc := &serviceDevice{manager: nil}
	_, err := svc.GetDeviceWebhook(context.Background(), "test-device")
	if err == nil {
		t.Fatal("expected error when manager is nil")
	}
	if err.Error() != "device manager not initialized" {
		t.Fatalf("unexpected error: %v", err)
	}
}
