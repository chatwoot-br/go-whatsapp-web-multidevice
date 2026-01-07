# Plan: Add Helm Chart for go-whatsapp-web-multidevice

## Overview

Create a production-ready Helm chart to deploy the WhatsApp Web API server on Kubernetes. The chart will support both REST API and MCP modes, configurable storage backends (SQLite/PostgreSQL), and multi-device management.

## Chart Structure

```
charts/gowa/
├── Chart.yaml              # Chart metadata (name, version, appVersion)
├── values.yaml             # Default configuration values
├── templates/
│   ├── _helpers.tpl        # Template helpers (labels, names, selectors)
│   ├── deployment.yaml     # Main application deployment
│   ├── service.yaml        # ClusterIP/NodePort/LoadBalancer service
│   ├── ingress.yaml        # Optional ingress resource
│   ├── configmap.yaml      # Non-sensitive configuration
│   ├── secret.yaml         # Sensitive data (webhooks, auth, db creds)
│   ├── pvc.yaml            # Persistent volume claims for storage
│   ├── serviceaccount.yaml # Optional service account
│   └── NOTES.txt           # Post-install usage notes
└── README.md               # Chart documentation
```

## Files to Create

### 1. `charts/gowa/Chart.yaml`
- Chart name: `gowa`
- API version: v2 (Helm 3)
- App version: v8.1.0 (from current release)
- Keywords, maintainers, sources

### 2. `charts/gowa/values.yaml`
Key configuration sections:
- **replicaCount**: Default 1 (SQLite limitation)
- **image**:
  - repository: `ghcr.io/chatwoot-br/go-whatsapp-web-multidevice`
  - tag: defaults to appVersion
  - pullPolicy: IfNotPresent
- **mode**: `rest` or `mcp` (runtime mode)
- **service**: Type, ports (3000 for REST, 8080 for MCP)
- **ingress**: Enabled, className, hosts, TLS
- **persistence**: Single volume mounted at `/app` for all data (storages/, statics/)
- **config**: App settings (port, host, debug, basePath, trustedProxies)
- **whatsapp**: WhatsApp settings (autoReply, autoMarkRead, autoDownloadMedia, accountValidation, chatStorage)
- **webhook**: URL, secret, events, insecureSkipVerify
- **auth**: Basic auth credentials
- **database**: URI settings for main DB and keys DB
- **resources**: CPU/memory limits and requests
- **probes**: Liveness and readiness probe configuration
- **nodeSelector, tolerations, affinity**: Scheduling options

### 3. `templates/_helpers.tpl`
- `gowa.name`: Chart name
- `gowa.fullname`: Release fullname
- `gowa.labels`: Standard Kubernetes labels
- `gowa.selectorLabels`: Pod selector labels
- `gowa.serviceAccountName`: SA name helper

### 4. `templates/deployment.yaml`
- Single replica deployment (SQLite constraint)
- Container with configurable command (`rest` or `mcp`)
- Environment variables from ConfigMap and Secret
- Volume mounts for persistent data and media
- Liveness/readiness probes on health endpoint
- Resource limits and requests
- Security context (non-root user)

### 5. `templates/service.yaml`
- Configurable service type (ClusterIP, NodePort, LoadBalancer)
- Port mapping based on mode (3000 for REST, 8080 for MCP)

### 6. `templates/ingress.yaml`
- Optional ingress (controlled by `ingress.enabled`)
- Support for ingress class, annotations, TLS
- Path-based routing with basePath support

### 7. `templates/configmap.yaml`
Non-sensitive configuration:
- APP_PORT, APP_HOST, APP_DEBUG, APP_OS
- APP_BASE_PATH, APP_TRUSTED_PROXIES
- WHATSAPP_AUTO_REPLY, WHATSAPP_AUTO_MARK_READ
- WHATSAPP_AUTO_DOWNLOAD_MEDIA, WHATSAPP_ACCOUNT_VALIDATION
- WHATSAPP_CHAT_STORAGE, WHATSAPP_WEBHOOK_EVENTS

### 8. `templates/secret.yaml`
Sensitive configuration:
- APP_BASIC_AUTH
- WHATSAPP_WEBHOOK (URLs)
- WHATSAPP_WEBHOOK_SECRET
- DB_URI, DB_KEYS_URI (if PostgreSQL)

### 9. `templates/pvc.yaml`
Single PVC (if persistence enabled):
- Mounted at `/app/storages` for databases and `/app/statics` for media files
- Uses subPath mounts from one volume

### 10. `templates/NOTES.txt`
Post-install instructions:
- How to access the service
- How to get the QR code for login
- WebSocket connection info
- Webhook configuration notes

### 11. `README.md`
- Prerequisites
- Installation instructions
- Configuration reference
- Upgrading notes
- Uninstallation

## Key Design Decisions

1. **Chart Location**: `charts/gowa` - standard structure allowing future charts.

2. **Single Replica Default**: SQLite doesn't support concurrent writes, so default to 1 replica. Document PostgreSQL connection for HA setups.

3. **External Database Only**: No PostgreSQL subchart dependency - users provide their own connection string if needed.

4. **Single PVC**: One volume for all persistent data (databases + media) with subPath mounts for simplicity.

5. **Mode Selection**: Use `mode` value to switch between REST and MCP at deployment time.

6. **Health Probes**: Use HTTP probe on `/` for REST mode (Fiber serves index).

7. **Keep It Simple**: Core resources only (Deployment, Service, Ingress, ConfigMap, Secret, PVC). No NetworkPolicy, PDB, or HPA.

## Verification

1. **Lint the chart**: `helm lint charts/gowa`
2. **Template rendering**: `helm template test charts/gowa`
3. **Dry-run install**: `helm install --dry-run --debug test charts/gowa`
4. **Local testing**: Deploy to minikube/kind and verify:
   - Service is accessible
   - QR code login works
   - Webhook delivery functions
   - Data persists across pod restarts
