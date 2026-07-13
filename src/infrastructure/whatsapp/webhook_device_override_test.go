package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
)

// signatureFor recomputes the X-Hub-Signature-256 body signature for a secret.
func signatureFor(t *testing.T, secret string, body []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// A device webhook config is authoritative for the URL it owns — including its zero
// values. With a global secret set and the device's own secret EMPTY, the delivery to the
// device URL must be signed with the device's (empty) secret, not the global one: a
// consumer reading its empty secret from GET /devices/{id}/webhook otherwise cannot
// validate the HMAC without being handed the unrelated global key.
func TestSubmitWebhook_DeviceConfigIsAuthoritativeForTheSecret(t *testing.T) {
	origSecret := config.WhatsappWebhookSecret
	defer func() { config.WhatsappWebhookSecret = origSecret }()
	config.WhatsappWebhookSecret = "global-secret"

	type delivery struct {
		signature string
		body      []byte
	}
	delivered := make(chan delivery, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		delivered <- delivery{signature: r.Header.Get("X-Hub-Signature-256"), body: body}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	payload := map[string]any{"event": "message"}
	url := srv.URL
	deviceCfg := &domainChatStorage.DeviceWebhookConfig{WebhookURL: &url} // secret deliberately empty

	if err := submitWebhook(context.Background(), payload, srv.URL, deviceCfg); err != nil {
		t.Fatalf("submitWebhook: %v", err)
	}

	got := <-delivered
	sig := got.signature
	// Sign the body the server actually received, so the assertion is about the SECRET,
	// not about JSON encoding.
	body := got.body
	if len(body) == 0 {
		body, _ = json.Marshal(payload)
	}
	if want := signatureFor(t, "global-secret", body); sig == want {
		t.Fatal("device delivery was signed with the GLOBAL secret; the device config must own the secret for its own URL")
	}
	if want := signatureFor(t, "", body); sig != want {
		t.Fatalf("expected the device's (empty) secret to sign its delivery\n got: %s\nwant: %s", sig, want)
	}
}

// Mirror for TLS: with the global insecure-skip flag ON, a device webhook that leaves
// webhook_insecure_skip_verify false must get real certificate verification back for its
// own URL. Verified behaviorally: the request to an httptest TLS server with an untrusted
// self-signed cert must FAIL for the device config (verification on) and SUCCEED under the
// global flag (verification off).
func TestSubmitWebhook_DeviceConfigCanRestoreTLSVerification(t *testing.T) {
	origSkip, origSecret := config.WhatsappWebhookInsecureSkipVerify, config.WhatsappWebhookSecret
	defer func() {
		config.WhatsappWebhookInsecureSkipVerify, config.WhatsappWebhookSecret = origSkip, origSecret
	}()
	config.WhatsappWebhookInsecureSkipVerify = true // global: skip verification
	config.WhatsappWebhookSecret = ""

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	payload := map[string]any{"event": "message"}

	// No device config -> global flag applies -> untrusted cert accepted.
	if err := submitWebhook(context.Background(), payload, srv.URL, nil); err != nil {
		t.Fatalf("expected the global skip-verify flag to accept the self-signed cert, got: %v", err)
	}

	// Device config with skip=false -> verification restored -> untrusted cert rejected.
	url := srv.URL
	deviceCfg := &domainChatStorage.DeviceWebhookConfig{WebhookURL: &url, WebhookInsecureSkipVerify: false}
	if err := submitWebhook(context.Background(), payload, srv.URL, deviceCfg); err == nil {
		t.Fatal("expected TLS verification to be restored for a device webhook with webhook_insecure_skip_verify=false, but the untrusted cert was accepted")
	}
}
