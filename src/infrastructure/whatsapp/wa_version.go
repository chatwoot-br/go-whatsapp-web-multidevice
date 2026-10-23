package whatsapp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
)

// waVersionFetchTimeout bounds the startup lookup of the current WhatsApp Web
// client version. Startup blocks for at most this long when web.whatsapp.com is
// unreachable; the embedded version is used after that.
const waVersionFetchTimeout = 10 * time.Second

// RefreshWAVersion selects the startup WhatsApp Web client version and logs the
// outcome. See readme.md's Configuration section for the startup lookup contract.
//
// It must run before any client connects: store.SetWAVersion mutates the
// client payload shared by every whatsmeow client in the process.
func RefreshWAVersion(ctx context.Context) {
	embedded := store.GetWAVersion()
	httpClient, err := waVersionHTTPClient(config.WhatsappProxyURL)
	if err != nil {
		logrus.Warnf("WhatsApp Web client version: using embedded %s (invalid proxy URL: %v)", embedded, utils.RedactURLError(err, config.WhatsappProxyURL))
		return
	}

	ctx, cancel := context.WithTimeout(ctx, waVersionFetchTimeout)
	defer cancel()

	latest, err := resolveWAVersion(ctx, httpClient, embedded)
	if err != nil {
		logrus.Warnf("WhatsApp Web client version: using embedded %s (refresh failed: %v)", embedded, utils.RedactURLError(err, config.WhatsappProxyURL))
		return
	}
	if !embedded.LessThan(latest) {
		logrus.Infof("WhatsApp Web client version: using embedded %s (web.whatsapp.com reports %s, not newer)", embedded, latest)
		return
	}
	store.SetWAVersion(latest)
	logrus.Infof("WhatsApp Web client version: using %s from web.whatsapp.com (embedded %s)", latest, embedded)
}

// resolveWAVersion fetches the version web.whatsapp.com currently serves. It
// never returns a zero version: a response that parses but carries no usable
// version is reported as an error so the caller falls back to embedded.
func resolveWAVersion(ctx context.Context, httpClient *http.Client, embedded store.WAVersionContainer) (store.WAVersionContainer, error) {
	latest, err := whatsmeow.GetLatestVersion(ctx, httpClient)
	if err != nil {
		return embedded, err
	}
	// GetLatestVersion fills {2, 3000, revision}; a zero revision means the page
	// parsed but carried no real build number.
	if latest == nil || latest.IsZero() || latest[2] == 0 {
		return embedded, fmt.Errorf("web.whatsapp.com returned an empty version")
	}
	return *latest, nil
}

// waVersionHTTPClient returns the HTTP client for the version lookup. With a
// WhatsApp proxy configured, the lookup goes through it like the pre-login
// websocket does, so egress-restricted deployments still reach web.whatsapp.com.
func waVersionHTTPClient(proxyURL string) (*http.Client, error) {
	if proxyURL == "" {
		return http.DefaultClient, nil
	}
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(parsed)
	return &http.Client{Transport: transport}, nil
}
