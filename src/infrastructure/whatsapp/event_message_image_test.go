package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Without a webhook or Chatwoot consumer, receipt must only store the message
// metadata. The duplicate-download regression would attempt a download using
// this disconnected client and emit a real download error. This checks that
// observable failure, rather than checking which helper functions exist.
func TestHandleMessageImageWithoutForwardingDoesNotDownload(t *testing.T) {
	originalLog := log
	originalMedia, originalStorages := config.PathMedia, config.PathStorages
	originalDownload := config.WhatsappAutoDownloadMedia
	originalWebhooks, originalChatwoot := config.WhatsappWebhook, config.ChatwootEnabled
	originalReply, originalMarkRead := config.WhatsappAutoReplyMessage, config.WhatsappAutoMarkRead
	defer func() {
		log = originalLog
		config.PathMedia, config.PathStorages = originalMedia, originalStorages
		config.WhatsappAutoDownloadMedia = originalDownload
		config.WhatsappWebhook, config.ChatwootEnabled = originalWebhooks, originalChatwoot
		config.WhatsappAutoReplyMessage, config.WhatsappAutoMarkRead = originalReply, originalMarkRead
	}()
	config.WhatsappWebhook = nil
	config.ChatwootEnabled = false
	config.WhatsappAutoReplyMessage = ""
	config.WhatsappAutoMarkRead = false

	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("auto_download_%t", enabled), func(t *testing.T) {
			config.WhatsappAutoDownloadMedia = enabled
			root := t.TempDir()
			config.PathMedia = filepath.Join(root, "statics", "media")
			config.PathStorages = filepath.Join(root, "storages")
			for _, dir := range []string{config.PathMedia, config.PathStorages} {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			logger := &imageReceiptErrorLogger{Logger: waLog.Noop}
			log = logger
			client := whatsmeow.NewClient(&store.Device{}, waLog.Noop)
			repo := &messageHandlerRepoSpy{}
			jid := types.NewJID("15551234567", types.DefaultUserServer)
			evt := &events.Message{
				Info: types.MessageInfo{
					MessageSource: types.MessageSource{Chat: jid, Sender: jid},
					ID:            "image-without-forwarding",
					Timestamp:     time.Now(),
				},
				Message: &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
					Caption:    protoString("image caption"),
					Mimetype:   protoString("image/jpeg"),
					DirectPath: protoString("/disposable-image"),
				}},
			}
			handler(context.Background(), NewDeviceInstance("image-test", client, repo), evt)
			if got := repo.createMessageCount(); got != 1 {
				t.Fatalf("image metadata stored %d times, want once", got)
			}
			if len(logger.errors) != 0 {
				t.Errorf("image receipt attempted unnecessary work: %v", logger.errors)
			}
			for _, dir := range []string{config.PathMedia, config.PathStorages} {
				entries, err := os.ReadDir(dir)
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) != 0 {
					t.Errorf("image receipt created files in %s: %v", dir, entries)
				}
			}
		})
	}
}

type imageReceiptErrorLogger struct {
	waLog.Logger
	errors []string
}

func (l *imageReceiptErrorLogger) Errorf(format string, args ...any) {
	l.errors = append(l.errors, fmt.Sprintf(format, args...))
}

// The public webhook contract must retain image URLs/captions when download is
// disabled, and still deliver the caption when attachment metadata is unusable.
// Receive the actual HTTP POST instead of replacing the webhook submitter.
func TestForwardImageDownloadModesPreserveWebhook(t *testing.T) {
	originalDownload, originalWebhooks := config.WhatsappAutoDownloadMedia, config.WhatsappWebhook
	originalEvents, originalChatwoot := config.WhatsappWebhookEvents, config.ChatwootEnabled
	defer func() {
		config.WhatsappAutoDownloadMedia, config.WhatsappWebhook = originalDownload, originalWebhooks
		config.WhatsappWebhookEvents, config.ChatwootEnabled = originalEvents, originalChatwoot
	}()
	config.WhatsappWebhookEvents = nil
	config.ChatwootEnabled = false
	client := whatsmeow.NewClient(&store.Device{}, waLog.Noop)

	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("auto_download_%t", enabled), func(t *testing.T) {
			config.WhatsappAutoDownloadMedia = enabled
			received := make(chan []byte, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil || r.Method != http.MethodPost {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				received <- body
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			config.WhatsappWebhook = []string{server.URL}
			jid := types.NewJID("15551234567", types.DefaultUserServer)
			evt := &events.Message{
				Info: types.MessageInfo{
					MessageSource: types.MessageSource{Chat: jid, Sender: jid},
					ID:            "webhook-image-test",
					Timestamp:     time.Date(2026, time.October, 14, 12, 0, 0, 0, time.UTC),
				},
				Message: &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
					URL:      protoString("https://example.invalid/image.jpg"),
					Caption:  protoString("preserve this caption"),
					Mimetype: protoString("image/jpeg"),
				}},
			}
			if err := forwardMessageToWebhook(context.Background(), client, evt, nil); err != nil {
				t.Fatal(err)
			}
			select {
			case body := <-received:
				t.Logf("received HTTP webhook: %s", body)
				var event WebhookEvent
				if err := json.Unmarshal(body, &event); err != nil {
					t.Fatal(err)
				}
				if event.Event != "message" || event.Payload["id"] != "webhook-image-test" || event.Payload["body"] != "preserve this caption" {
					t.Fatalf("image message/caption lost: %s", body)
				}
				if enabled {
					if _, exists := event.Payload["image"]; exists {
						t.Fatalf("unavailable attachment emitted as a valid image: %s", body)
					}
				} else {
					image, ok := event.Payload["image"].(map[string]any)
					if !ok || image["url"] != "https://example.invalid/image.jpg" || image["caption"] != "preserve this caption" {
						t.Fatalf("URL/caption contract broken: %s", body)
					}
				}
			case <-time.After(2 * time.Second):
				t.Fatal("no HTTP webhook received")
			}
		})
	}
}
