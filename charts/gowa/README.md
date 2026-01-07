# Gowa Helm Chart

A Helm chart for deploying [go-whatsapp-web-multidevice](https://github.com/chatwoot-br/go-whatsapp-web-multidevice) - a WhatsApp Web API server supporting both REST API and MCP modes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- PV provisioner support in the underlying infrastructure (if persistence is enabled)

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
helm install my-release ./charts/gowa
```

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
helm delete my-release
```

## Configuration

The following table lists the configurable parameters of the Gowa chart and their default values.

### Global Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas (use 1 for SQLite) | `1` |
| `mode` | Application mode: `rest` or `mcp` | `rest` |

### Image Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Image repository | `ghcr.io/chatwoot-br/go-whatsapp-web-multidevice` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (defaults to appVersion) | `""` |
| `imagePullSecrets` | Image pull secrets | `[]` |

### Service Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `3000` |

### Ingress Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.annotations` | Ingress annotations | `{}` |
| `ingress.hosts` | Ingress hosts configuration | See values.yaml |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### Persistence Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.enabled` | Enable persistence | `true` |
| `persistence.storageClass` | Storage class (empty uses default) | `""` |
| `persistence.accessMode` | PVC access mode | `ReadWriteOnce` |
| `persistence.size` | PVC size | `5Gi` |

### Application Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.port` | Application port (APP_PORT) | `"3000"` |
| `config.host` | Bind address (APP_HOST) | `"0.0.0.0"` |
| `config.debug` | Debug mode (APP_DEBUG) | `"false"` |
| `config.os` | Device name (APP_OS) | `"Chrome"` |
| `config.basePath` | Base path (APP_BASE_PATH) | `""` |
| `config.trustedProxies` | Trusted proxies (APP_TRUSTED_PROXIES) | `"0.0.0.0/0"` |

### WhatsApp Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `whatsapp.autoReply` | Auto-reply message | `""` |
| `whatsapp.autoMarkRead` | Auto-mark messages as read | `"false"` |
| `whatsapp.autoDownloadMedia` | Auto-download media | `"true"` |
| `whatsapp.accountValidation` | Account validation | `"true"` |
| `whatsapp.chatStorage` | Chat history storage | `"true"` |

### Webhook Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `webhook.urls` | Webhook URLs (comma-separated) | `""` |
| `webhook.secret` | Webhook signing secret | `""` |
| `webhook.insecureSkipVerify` | Skip TLS verification | `"false"` |
| `webhook.events` | Webhook events | See values.yaml |

### Authentication

| Parameter | Description | Default |
|-----------|-------------|---------|
| `auth.basicAuth` | Basic auth (user1:pass1,user2:pass2) | `""` |

### Database Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `database.uri` | Main database URI | `"file:storages/whatsapp.db?_foreign_keys=on"` |
| `database.keysUri` | Keys database URI | `"file::memory:?cache=shared&_foreign_keys=on"` |

### Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources` | CPU/memory resources | `{}` |
| `livenessProbe` | Liveness probe configuration | See values.yaml |
| `readinessProbe` | Readiness probe configuration | See values.yaml |

## Examples

### Basic Installation

```bash
helm install gowa ./charts/gowa
```

### With Ingress

```bash
helm install gowa ./charts/gowa \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=whatsapp.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

### With Webhook

```bash
helm install gowa ./charts/gowa \
  --set webhook.urls=https://example.com/webhook \
  --set webhook.secret=my-secret-key
```

### With Basic Auth

```bash
helm install gowa ./charts/gowa \
  --set auth.basicAuth=admin:password123
```

### With PostgreSQL (for HA)

```bash
helm install gowa ./charts/gowa \
  --set replicaCount=2 \
  --set database.uri="postgres://user:pass@postgres:5432/whatsapp?sslmode=disable"
```

### MCP Mode

```bash
helm install gowa ./charts/gowa \
  --set mode=mcp \
  --set service.port=8080
```

## Upgrading

```bash
helm upgrade my-release ./charts/gowa
```

## Notes

- **Single Replica**: When using SQLite (default), use only 1 replica as SQLite doesn't support concurrent writes.
- **PostgreSQL**: For high-availability deployments, configure an external PostgreSQL database.
- **Persistence**: Data is stored in a single PVC with subPath mounts for `/app/storages` and `/app/statics`.
- **Modes**: The application supports two modes:
  - `rest`: REST API server (default, port 3000)
  - `mcp`: Model Context Protocol server (port 8080)
