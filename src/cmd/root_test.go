package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactSettings(t *testing.T) {
	settings := map[string]any{
		"app_port":                "3000",
		"app_basic_auth":          "user1:pass1,user2:pass2",
		"db_uri":                  "postgres://gowa:dbpass@db.example.com:5432/gowa?sslmode=disable",
		"db_keys_uri":             "file:storages/whatsapp.db?_foreign_keys=on",
		"whatsapp_proxy_url":      "socks5://proxyuser:proxypass@proxy.example.com:1080",
		"whatsapp_webhook":        "https://hookuser:hookpass@a.example.com/hook,https://b.example.com/hook,https://u:2024,hookpass2@c.example.com",
		"chatwoot_import_db_uri":  "pguser:pgpass@db.example.com:5432/chatwoot",
		"whatsapp_webhook_secret": "super-secret-key",
		"chatwoot_api_token":      "tok-123",
		"chatwoot_webhook_secret": "",
		"chatwoot_ignore_jids":    "123@s.whatsapp.net,456@g.us",
		"whatsapp_auto_reply":     "Auto reply message",
		"app_debug":               true,
		"nested":                  map[string]any{"proxy_url": "socks5://nesteduser:nestedpass@proxy.example.com:1080", "api_token": "tok-456"},
	}

	got := redactSettings(settings)

	assert.Equal(t, map[string]any{
		"app_port":                "3000",
		"app_basic_auth":          "***",
		"db_uri":                  "postgres://***:***@db.example.com:5432/gowa?sslmode=disable",
		"db_keys_uri":             "file:storages/whatsapp.db?_foreign_keys=on",
		"whatsapp_proxy_url":      "socks5://***:***@proxy.example.com:1080",
		"whatsapp_webhook":        "https://***:***@a.example.com/hook,https://b.example.com/hook,https://***:***@c.example.com",
		"chatwoot_import_db_uri":  "***:***@db.example.com:5432/chatwoot",
		"whatsapp_webhook_secret": "***",
		"chatwoot_api_token":      "***",
		"chatwoot_webhook_secret": "",
		"chatwoot_ignore_jids":    "123@s.whatsapp.net,456@g.us",
		"whatsapp_auto_reply":     "Auto reply message",
		"app_debug":               true,
		"nested":                  map[string]any{"proxy_url": "socks5://***:***@proxy.example.com:1080", "api_token": "***"},
	}, got)

	printed := fmt.Sprint(got)
	for _, secret := range []string{"pass1", "pass2", "dbpass", "proxyuser", "proxypass", "hookpass", "hookpass2", "pguser", "pgpass", "super-secret-key", "tok-123", "nesteduser", "nestedpass", "tok-456"} {
		assert.NotContains(t, printed, secret)
	}
	assert.Equal(t, "socks5://proxyuser:proxypass@proxy.example.com:1080", settings["whatsapp_proxy_url"], "input map must not be modified")
}

func TestRedactSettingsProxyURLEdgeCases(t *testing.T) {
	for raw, want := range map[string]string{
		"socks5://proxyuser:2024,proxypass@proxy.example.com:1080": "socks5://***:***@proxy.example.com:1080",
		"socks5://proxyuser:2024#proxypass@proxy.example.com:1080": "socks5://***:***@proxy.example.com:1080",
		"proxyuser:proxypass@proxy.example.com:1080":               "***:***@proxy.example.com:1080",
	} {
		got := redactSettings(map[string]any{"whatsapp_proxy_url": raw})
		assert.Equal(t, want, got["whatsapp_proxy_url"], raw)
	}
}
