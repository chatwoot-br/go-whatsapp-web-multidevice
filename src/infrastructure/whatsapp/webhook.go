package whatsapp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/sirupsen/logrus"
)

func submitWebhook(ctx context.Context, payload map[string]any, url string, webhookConfig *chatstorage.DeviceWebhookConfig) error {
	// Determine effective config - use device-specific if set, otherwise fall back to global.
	//
	// A non-nil webhookConfig means this delivery is going to the DEVICE's own URL (it is
	// only built when the device row carries a webhook_url, and it is what selects the
	// target URLs), so the device config is authoritative for that URL — including its
	// zero values. Merging field-by-field instead ("only override when truthy/non-empty")
	// silently lets global settings govern a URL they do not own:
	//   - webhook_insecure_skip_verify=false could never restore TLS verification while the
	//     global flag was true, so a self-signed global webhook disabled certificate checks
	//     for every device webhook as well;
	//   - an empty webhook_secret signed the device's deliveries with the GLOBAL secret, so
	//     a consumer reading its (empty) secret from GET /devices/{id}/webhook could not
	//     validate the HMAC without being handed the unrelated global key.
	// The global path (webhookConfig == nil) is unchanged.
	insecureSkipVerify := config.WhatsappWebhookInsecureSkipVerify
	webhookSecret := config.WhatsappWebhookSecret

	if webhookConfig != nil {
		insecureSkipVerify = webhookConfig.WebhookInsecureSkipVerify
		webhookSecret = webhookConfig.WebhookSecret
	}

	// Configure HTTP client with optional TLS skip verification
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	postBody, err := json.Marshal(payload)
	if err != nil {
		return pkgError.WebhookError(fmt.Sprintf("Failed to marshal body: %v", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return pkgError.WebhookError(fmt.Sprintf("error when create http object %v", err))
	}

	secretKey := []byte(webhookSecret)
	signature, err := utils.GetMessageDigestOrSignature(postBody, secretKey)
	if err != nil {
		return pkgError.WebhookError(fmt.Sprintf("error when create signature %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", fmt.Sprintf("sha256=%s", signature))

	var attempt int
	var maxAttempts = 5
	var sleepDuration = 1 * time.Second

	for attempt = 0; attempt < maxAttempts; attempt++ {
		// Create new request body for each attempt
		req.Body = io.NopCloser(bytes.NewBuffer(postBody))
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				logrus.Infof("Successfully submitted webhook on attempt %d", attempt+1)
				return nil
			}
			err = fmt.Errorf("webhook returned status %d", resp.StatusCode)
		}
		logrus.Warnf("Attempt %d to submit webhook failed: %v", attempt+1, err)
		if attempt < maxAttempts-1 {
			time.Sleep(sleepDuration)
			sleepDuration *= 2
		}
	}

	return pkgError.WebhookError(fmt.Sprintf("error when submit webhook after %d attempts: %v", attempt, err))
}
