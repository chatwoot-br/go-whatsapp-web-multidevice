# GOWA - Go WhatsApp Admin API Helm Chart

Helm chart for deploying the Go WhatsApp Web Multidevice Admin API on Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- Persistent Volume Provisioner (for production)

## Security Considerations

> ⚠️ **IMPORTANT**: Review these security settings before production deployment.

### Credentials

**Never use default credentials in production!**

```bash
# Production deployment with secure credentials
helm install gowa ./charts/gowa \
  --set admin.supervisor.password="$(openssl rand -base64 32)" \
  --set admin.token="$(openssl rand -base64 32)"
```

Or use existing secrets:

```bash
helm install gowa ./charts/gowa \
  --set admin.supervisor.existingSecret="my-supervisor-secret" \
  --set admin.existingSecret="my-admin-secret"
```

### Security Features

| Feature | Default | Production Recommendation |
|---------|---------|--------------------------|
| Non-root containers | ✅ Enabled | Keep enabled |
| Dropped capabilities | ✅ ALL dropped | Keep enabled |
| Read-only root filesystem | ❌ Disabled | Enable if possible |
| NetworkPolicy | ❌ Disabled | Enable for isolation |
| PodDisruptionBudget | ❌ Disabled | Enable for HA |

## Quick Start

```bash
# Install with default settings (development only)
helm install gowa ./charts/gowa

# Access via port-forward
kubectl port-forward svc/gowa 8088:8088 8080:8080

# Admin API: http://localhost:8088
# Swagger UI: http://localhost:8080
```

## Configuration

### Basic Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Container image | `ghcr.io/chatwoot-br/go-whatsapp-web-multidevice` |
| `image.tag` | Image tag | Chart appVersion |

### Admin API

| Parameter | Description | Default |
|-----------|-------------|---------|
| `admin.port` | Admin API port | `8088` |
| `admin.token` | Admin token (creates secret) | `""` |
| `admin.existingSecret` | Use existing secret | `""` |
| `admin.supervisor.username` | Supervisor username | `"admin"` |
| `admin.supervisor.password` | Supervisor password | `""` |
| `admin.supervisor.existingSecret` | Use existing secret | `""` |

### Network & Security

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podDisruptionBudget.enabled` | Enable PDB | `false` |
| `podDisruptionBudget.minAvailable` | Minimum available | `1` |
| `networkPolicy.enabled` | Enable NetworkPolicy | `false` |

### Persistence

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.enabled` | Enable PVC | `true` |
| `persistence.size` | Storage size | `10Gi` |
| `persistence.accessMode` | Access mode | `ReadWriteOnce` |

## Architecture

The chart deploys:

```
┌─────────────────────────────────────────────────────────┐
│                         Pod                             │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │  Init:      │  │  Init:      │  │                 │  │
│  │  Directories│→ │  Config     │→ │   Containers    │  │
│  │  (busybox)  │  │  Generator  │  │                 │  │
│  └─────────────┘  └─────────────┘  └─────────────────┘  │
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │  Admin API  │  │ Supervisord │  │   Swagger UI    │  │
│  │  :8088      │← │  :9001      │  │   :8080         │  │
│  └─────────────┘  └─────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

- **Init Containers**: Create directories and generate config with secrets
- **Admin API**: REST API for managing WhatsApp instances
- **Supervisord**: Process manager for WhatsApp instance workers
- **Swagger UI**: Interactive API documentation (optional)

## Ports

| Port | Name | Description |
|------|------|-------------|
| 8088 | admin-api | Admin REST API |
| 8080 | swagger-ui | Swagger documentation |
| 9001 | supervisor-rpc | Supervisord RPC (internal) |
| 3001-3010 | whatsapp-* | WhatsApp instance ports |

## Health Checks

```bash
# Liveness
curl http://localhost:8088/healthz

# Readiness
curl http://localhost:8088/readyz
```

## Troubleshooting

### Check pod status

```bash
kubectl get pods -l app.kubernetes.io/name=gowa
kubectl describe pod <pod-name>
```

### View logs

```bash
# Admin API
kubectl logs <pod-name> -c gowa -f

# Supervisord
kubectl logs <pod-name> -c supervisord -f

# Init containers
kubectl logs <pod-name> -c init-directories
kubectl logs <pod-name> -c init-supervisor-config
```

### Common issues

1. **401 Unauthorized from Supervisord**: Check that `SUPERVISOR_USER` and `SUPERVISOR_PASS` match the config
2. **Slow startup**: Adjust probe `initialDelaySeconds` if needed
3. **Permission denied**: Verify `fsGroup` and `runAsUser` match the image requirements

## Upgrading

```bash
helm upgrade gowa ./charts/gowa --reuse-values
```

## Uninstalling

```bash
helm uninstall gowa
kubectl delete pvc -l app.kubernetes.io/name=gowa  # If you want to delete data
```

## Documentation

- [Full Documentation](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/tree/main/docs)
- [Kubernetes Deployment Guide](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/tree/main/docs/guides/deployment/kubernetes.md)
- [Admin API Reference](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/tree/main/docs/reference/api/admin-api-openapi.yaml)
