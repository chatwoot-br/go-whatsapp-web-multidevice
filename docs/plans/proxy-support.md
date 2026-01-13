# Plan: Add Proxy Support for WhatsApp Connections

## Status: Completed

## Summary

Add proxy configuration for WhatsApp connections with support for HTTP, HTTPS, and SOCKS5 proxies. Includes application code changes, Helm chart configuration, and UI display of the proxy IP address.

## Features Implemented

1. **Proxy configuration** via environment variables and CLI flags
2. **SOCKS5 and HTTP/HTTPS proxy support** for WhatsApp WebSocket and media connections
3. **Proxy options**: NoWebsocket, OnlyLogin, NoMedia
4. **Helm chart support** for Kubernetes deployments
5. **UI display** of the external IP address when using a proxy

---

## Discovery

### whatsmeow Proxy Support

The whatsmeow library supports proxy configuration via:
- `SetProxyAddress(addr string)` - Parses URL and sets proxy (HTTP/HTTPS/SOCKS5)
- `SetProxy(proxy Proxy)` - Custom proxy function
- `SetSOCKSProxy(px proxy.Dialer)` - SOCKS5 dialer
- Environment variable: `https_proxy` (read by default)

**SetProxyOptions flags:**
- `NoWebsocket` - Don't use proxy for websocket
- `OnlyLogin` - Use proxy only for pre-login websocket
- `NoMedia` - Don't use proxy for media uploads/downloads

---

## Implementation

### Phase 1: Application Code Changes

#### 1.1 Proxy config in settings
**File:** `src/config/settings.go`

```go
// Proxy configuration for WhatsApp connections
WhatsappProxyURL         string  // Proxy URL (http://, https://, socks5://)
WhatsappProxyNoWebsocket = false // Don't use proxy for websocket connections
WhatsappProxyOnlyLogin   = false // Use proxy only for pre-login websocket
WhatsappProxyNoMedia     = false // Don't use proxy for media uploads/downloads
```

#### 1.2 CLI flags and env bindings
**File:** `src/cmd/root.go`

Flags added:
- `--whatsapp-proxy-url` / `WHATSAPP_PROXY_URL`
- `--whatsapp-proxy-no-websocket` / `WHATSAPP_PROXY_NO_WEBSOCKET`
- `--whatsapp-proxy-only-login` / `WHATSAPP_PROXY_ONLY_LOGIN`
- `--whatsapp-proxy-no-media` / `WHATSAPP_PROXY_NO_MEDIA`

#### 1.3 Proxy applied to WhatsApp client
**File:** `src/infrastructure/whatsapp/init.go`

```go
// Configure proxy if specified
if config.WhatsappProxyURL != "" {
    proxyOpts := whatsmeow.SetProxyOptions{
        NoWebsocket: config.WhatsappProxyNoWebsocket,
        OnlyLogin:   config.WhatsappProxyOnlyLogin,
        NoMedia:     config.WhatsappProxyNoMedia,
    }
    if err := client.SetProxyAddress(config.WhatsappProxyURL, proxyOpts); err != nil {
        log.Warnf("Failed to set proxy: %v", err)
    } else {
        log.Infof("Proxy configured: %s", config.WhatsappProxyURL)
    }
}
```

#### 1.4 Proxy applied to device manager
**File:** `src/infrastructure/whatsapp/device_manager.go`

Same proxy configuration in `EnsureClient()` method for multi-device support.

#### 1.5 Environment example
**File:** `src/.env.example`

```bash
# WhatsApp Proxy Settings (optional)
# Proxy URL for WhatsApp connections (supports http://, https://, socks5://)
# WHATSAPP_PROXY_URL=socks5://user:pass@proxy.example.com:1080
# Don't use proxy for websocket connections
# WHATSAPP_PROXY_NO_WEBSOCKET=false
# Use proxy only for pre-login websocket
# WHATSAPP_PROXY_ONLY_LOGIN=false
# Don't use proxy for media uploads/downloads
# WHATSAPP_PROXY_NO_MEDIA=false
```

---

### Phase 2: Helm Chart Changes

#### 2.1 values.yaml
**File:** `charts/gowa/values.yaml`

```yaml
whatsapp:
  # proxy contains proxy configuration for WhatsApp connections
  proxy:
    # url is the proxy URL (WHATSAPP_PROXY_URL)
    # Supports http://, https://, socks5://
    # Example: socks5://user:pass@proxy.example.com:1080
    url: ""
    # noWebsocket disables proxy for websocket connections
    noWebsocket: "false"
    # onlyLogin uses proxy only for pre-login websocket
    onlyLogin: "false"
    # noMedia disables proxy for media uploads/downloads
    noMedia: "false"
```

#### 2.2 configmap.yaml
**File:** `charts/gowa/templates/configmap.yaml`

```yaml
{{- if .Values.whatsapp.proxy.url }}
WHATSAPP_PROXY_NO_WEBSOCKET: {{ .Values.whatsapp.proxy.noWebsocket | quote }}
WHATSAPP_PROXY_ONLY_LOGIN: {{ .Values.whatsapp.proxy.onlyLogin | quote }}
WHATSAPP_PROXY_NO_MEDIA: {{ .Values.whatsapp.proxy.noMedia | quote }}
{{- end }}
```

#### 2.3 secret.yaml
**File:** `charts/gowa/templates/secret.yaml`

```yaml
{{- if .Values.whatsapp.proxy.url }}
WHATSAPP_PROXY_URL: {{ .Values.whatsapp.proxy.url | quote }}
{{- end }}
```

#### 2.4 README.md
**File:** `charts/gowa/README.md`

Added proxy parameters documentation and usage examples.

---

### Phase 3: UI Proxy IP Display

#### 3.1 Device domain model
**File:** `src/domains/device/device.go`

Added `ProxyIP` field to display external IP when using proxy:
```go
ProxyIP string `json:"proxy_ip,omitempty"`
```

#### 3.2 Proxy IP lookup
**File:** `src/infrastructure/whatsapp/device_instance.go`

Added `FetchProxyIP()` method that:
1. Creates HTTP client with configured proxy (SOCKS5 or HTTP)
2. Fetches external IP from `https://api.ipify.org`
3. Caches the result to avoid repeated lookups

#### 3.3 Usecase integration
**File:** `src/usecase/device.go`

Updated `convertInstance()` to call `FetchProxyIP()` and include in response.

#### 3.4 UI display
**File:** `src/views/components/DeviceManager.js`

```html
<span v-if="dev.proxy_ip"> · IP: {{ dev.proxy_ip }}</span>
```

Display in device card: `State: logged_in · JID: xxx@s.whatsapp.net · IP: 123.45.67.89`

---

## Files Modified

### Application Code
| File | Change |
|------|--------|
| `src/config/settings.go` | Added proxy config fields |
| `src/cmd/root.go` | Added CLI flags and viper bindings |
| `src/infrastructure/whatsapp/init.go` | Applied proxy to client |
| `src/infrastructure/whatsapp/device_manager.go` | Applied proxy to device manager |
| `src/infrastructure/whatsapp/device_instance.go` | Added ProxyIP field and FetchProxyIP() |
| `src/domains/device/device.go` | Added ProxyIP to Device struct |
| `src/usecase/device.go` | Added proxy IP to device response |
| `src/views/components/DeviceManager.js` | Display proxy IP in UI |
| `src/.env.example` | Documented proxy env vars |

### Helm Chart
| File | Change |
|------|--------|
| `charts/gowa/values.yaml` | Added proxy section |
| `charts/gowa/templates/configmap.yaml` | Added proxy options |
| `charts/gowa/templates/secret.yaml` | Added proxy URL |
| `charts/gowa/README.md` | Documented proxy config |

---

## Usage

### Environment Variables

```bash
# SOCKS5 proxy
WHATSAPP_PROXY_URL=socks5://user:pass@proxy.example.com:1080

# HTTP proxy
WHATSAPP_PROXY_URL=http://proxy.example.com:8080

# Skip proxy for media (faster downloads)
WHATSAPP_PROXY_NO_MEDIA=true
```

### Helm Installation

```bash
# With SOCKS5 proxy
helm install gowa ./charts/gowa \
  --set whatsapp.proxy.url=socks5://user:pass@proxy:1080

# With HTTP proxy, media direct
helm install gowa ./charts/gowa \
  --set whatsapp.proxy.url=http://proxy:8080 \
  --set whatsapp.proxy.noMedia=true
```

### Proxy Type Recommendations

| Scenario | Recommended Proxy |
|----------|-------------------|
| General use | SOCKS5 |
| Corporate network (HTTP only) | HTTP |
| Bypassing geo-restrictions | SOCKS5 |
| Media-heavy usage | SOCKS5 + `noMedia=true` |

---

## Verification

1. **Go build**: `cd src && go build .`
2. **Helm lint**: `helm lint charts/gowa`
3. **Helm template**: `helm template test charts/gowa --set whatsapp.proxy.url=socks5://proxy:1080`
4. **Integration test**:
   - Run local SOCKS5 proxy: `docker run -d -p 1080:1080 serjs/go-socks5-proxy`
   - Set `WHATSAPP_PROXY_URL=socks5://localhost:1080`
   - Connect to WhatsApp and verify proxy IP shown in UI
