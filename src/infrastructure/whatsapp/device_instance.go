package whatsapp

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	domainDevice "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/device"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/net/proxy"
)

// DeviceInstance bundles a WhatsApp client with device metadata and scoped storage.
type DeviceInstance struct {
	mu              sync.RWMutex
	id              string
	client          *whatsmeow.Client
	chatStorageRepo domainChatStorage.IChatStorageRepository
	state           domainDevice.DeviceState
	displayName     string
	phoneNumber     string
	jid             string
	// lastJID mirrors devices.last_jid: the account a LOGGED-OUT slot still belongs to.
	// It is what distinguishes such a slot from a never-paired placeholder — both have an
	// empty live jid, but only the placeholder may adopt an unmatched store row on boot.
	lastJID     string
	proxyIP     string
	createdAt   time.Time
	onLoggedOut func(deviceID string) // Callback for remote logout cleanup

	// Pending passkey pairing state, populated by PairPasskey* events during login.
	passkeyChallenge     *types.WebAuthnPublicKey
	passkeyCode          string
	passkeySkipHandoffUX bool
}

func NewDeviceInstance(deviceID string, client *whatsmeow.Client, chatStorageRepo domainChatStorage.IChatStorageRepository) *DeviceInstance {
	jid := ""
	display := ""
	if client != nil && client.Store != nil && client.Store.ID != nil {
		jid = client.Store.ID.ToNonAD().String()
		display = client.Store.PushName
	}

	return &DeviceInstance{
		id:              deviceID,
		client:          client,
		chatStorageRepo: chatStorageRepo,
		state:           domainDevice.DeviceStateDisconnected,
		displayName:     display,
		jid:             jid,
		createdAt:       time.Now(),
	}
}

func (d *DeviceInstance) ID() string {
	return d.id
}

func (d *DeviceInstance) GetClient() *whatsmeow.Client {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.client
}

func (d *DeviceInstance) GetChatStorage() domainChatStorage.IChatStorageRepository {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.chatStorageRepo
}

func (d *DeviceInstance) SetState(state domainDevice.DeviceState) {
	d.mu.Lock()
	d.state = state
	d.mu.Unlock()
}

func (d *DeviceInstance) State() domainDevice.DeviceState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

func (d *DeviceInstance) DisplayName() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.displayName
}

func (d *DeviceInstance) PhoneNumber() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.phoneNumber
}

func (d *DeviceInstance) JID() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.jid
}

func (d *DeviceInstance) CreatedAt() time.Time {
	return d.createdAt
}

// SetClient attaches a WhatsApp client to this instance and updates metadata.
func (d *DeviceInstance) SetClient(client *whatsmeow.Client) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.client = client
	d.refreshIdentityLocked()
	d.state = domainDevice.DeviceStateDisconnected
}

// ResetClient detaches the WhatsApp client and clears the session-derived identity
// (jid, phone number) so the slot can be re-paired with a fresh client on the next
// login. The device id, display name and creation time are preserved, keeping the
// slot in place after a logout.
func (d *DeviceInstance) ResetClient() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.client = nil
	d.jid = ""
	d.phoneNumber = ""
	d.state = domainDevice.DeviceStateDisconnected
}

// LastJID returns the JID this slot was last paired to, retained through a keep-slot
// logout (which clears the live jid). A non-empty value means "logged out, but this slot
// belongs to that account" — as opposed to a never-paired placeholder, which has neither.
func (d *DeviceInstance) LastJID() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lastJID
}

// SetLastJID records the retained identity of a logged-out slot, mirroring the persisted
// devices.last_jid column in memory.
func (d *DeviceInstance) SetLastJID(jid string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastJID = jid
}

// SetChatStorage swaps the chat storage repository for this device.
func (d *DeviceInstance) SetChatStorage(repo domainChatStorage.IChatStorageRepository) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.chatStorageRepo = repo
}

// IsConnected returns the live connection flag if a client exists.
func (d *DeviceInstance) IsConnected() bool {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()
	if client == nil {
		return false
	}
	return client.IsConnected()
}

// IsLoggedIn returns the login status if a client exists.
func (d *DeviceInstance) IsLoggedIn() bool {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()
	if client == nil {
		return false
	}
	return client.IsLoggedIn()
}

// UpdateStateFromClient refreshes the snapshot state based on the client flags.
func (d *DeviceInstance) UpdateStateFromClient() domainDevice.DeviceState {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch {
	case d.client != nil && d.client.IsLoggedIn():
		d.state = domainDevice.DeviceStateLoggedIn
	case d.client != nil && d.client.IsConnected():
		d.state = domainDevice.DeviceStateConnected
	default:
		d.state = domainDevice.DeviceStateDisconnected
	}

	d.refreshIdentityLocked()
	return d.state
}

func (d *DeviceInstance) refreshIdentityLocked() {
	if d.client != nil && d.client.Store != nil && d.client.Store.ID != nil {
		d.jid = d.client.Store.ID.ToNonAD().String()
		d.displayName = d.client.Store.PushName
	}
}

// SetPasskeyChallenge stores a pending WebAuthn challenge and clears any previous confirmation code.
func (d *DeviceInstance) SetPasskeyChallenge(pk *types.WebAuthnPublicKey) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.passkeyChallenge = pk
	d.passkeyCode = ""
	d.passkeySkipHandoffUX = false
}

// SetPasskeyConfirmation stores the pairing confirmation code and clears the pending challenge.
func (d *DeviceInstance) SetPasskeyConfirmation(code string, skipHandoffUX bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.passkeyChallenge = nil
	d.passkeyCode = code
	d.passkeySkipHandoffUX = skipHandoffUX
}

// PasskeyState returns the pending challenge, confirmation code and skip-handoff flag.
func (d *DeviceInstance) PasskeyState() (*types.WebAuthnPublicKey, string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.passkeyChallenge, d.passkeyCode, d.passkeySkipHandoffUX
}

// ClearPasskeyState resets all pending passkey pairing state.
func (d *DeviceInstance) ClearPasskeyState() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.passkeyChallenge = nil
	d.passkeyCode = ""
	d.passkeySkipHandoffUX = false
}

// ClearPasskeyChallenge retires a consumed WebAuthn challenge WITHOUT touching the
// confirmation code. Callers that have just submitted the assertion must use this rather
// than ClearPasskeyState: PairPasskeyConfirmation can arrive before the submit call
// returns, and wiping all state would then discard the confirmation code that has already
// landed — leaving the pairing with no visible pending status and no code to confirm with.
func (d *DeviceInstance) ClearPasskeyChallenge() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.passkeyChallenge = nil
}

func (d *DeviceInstance) SetOnLoggedOut(callback func(deviceID string)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onLoggedOut = callback
}

// TriggerLoggedOut invokes the registered logout callback and reports whether one
// was wired — callers use a false return to run the cleanup themselves, so registry
// reconciliation never silently depends on in-memory wiring.
func (d *DeviceInstance) TriggerLoggedOut() bool {
	d.mu.RLock()
	callback := d.onLoggedOut
	deviceID := d.id
	d.mu.RUnlock()

	if callback == nil {
		return false
	}
	callback(deviceID)
	return true
}

// ProxyIP returns the cached external IP address when using a proxy.
func (d *DeviceInstance) ProxyIP() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.proxyIP
}

// FetchProxyIP fetches the external IP address through the configured proxy.
// Returns empty string if no proxy is configured or if the lookup fails.
func (d *DeviceInstance) FetchProxyIP() string {
	if config.WhatsappProxyURL == "" {
		return ""
	}

	// Check cache first
	d.mu.RLock()
	if d.proxyIP != "" {
		ip := d.proxyIP
		d.mu.RUnlock()
		return ip
	}
	d.mu.RUnlock()

	// Create HTTP client with proxy
	httpClient, err := createProxyHTTPClient(config.WhatsappProxyURL)
	if err != nil {
		return ""
	}

	// Fetch external IP from ipify
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org", nil)
	if err != nil {
		return ""
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	ip := strings.TrimSpace(string(body))

	// Cache the result
	d.mu.Lock()
	d.proxyIP = ip
	d.mu.Unlock()

	return ip
}

// createProxyHTTPClient creates an HTTP client configured to use the specified proxy.
func createProxyHTTPClient(proxyURL string) (*http.Client, error) {
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	var transport *http.Transport

	switch parsedURL.Scheme {
	case "socks5":
		// SOCKS5 proxy
		auth := &proxy.Auth{}
		if parsedURL.User != nil {
			auth.User = parsedURL.User.Username()
			auth.Password, _ = parsedURL.User.Password()
		}

		dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, auth, proxy.Direct)
		if err != nil {
			return nil, err
		}

		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		}

	case "http", "https":
		// HTTP proxy
		transport = &http.Transport{
			Proxy: http.ProxyURL(parsedURL),
		}

	default:
		return nil, nil
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}, nil
}
