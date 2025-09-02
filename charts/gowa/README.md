# Go WhatsApp Web Multidevice Admin API Helm Chart

This Helm chart deploys the Admin API for Go WhatsApp Web Multidevice, which allows you to manage multiple WhatsApp instances through a REST API.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- Persistent Volume Provisioner support in the underlying infrastructure

## Installing the Chart

To install the chart with the release name `whatsapp-admin`:

```bash
helm install whatsapp-admin ./charts
```

## Configuration

The following table lists the configurable parameters and their default values.

### Basic Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Container image repository | `ghcr.io/chatwoot-br/go-whatsapp-web-multidevice` |
| `image.tag` | Container image tag | `main` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |

### Admin API Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `admin.port` | Admin API server port | `8088` |
| `admin.token` | Admin token (creates secret) | `""` |
| `admin.existingSecret` | Existing secret name for admin token | `""` |
| `admin.existingSecretKey` | Key in existing secret | `"admin-token"` |

### Service Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `8088` |

### Ingress Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
# Go WhatsApp Web Multidevice — Admin API Helm Chart

This consolidated guide documents configuration, deployment, debugging and authentication guidance for the Admin API Helm chart used to manage multiple WhatsApp instances via Supervisord.

## Quick summary

- Chart name: `gowa`
- Image: `ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest`
- Primary ports: Admin API 8088, Supervisord RPC 9001, WhatsApp instances 3001-3010
- Mode: Admin API runs in one container and Supervisord runs as a sidecar. They share volumes for configs, logs and instances data.

## Table of Contents

1. Deployment overview
2. Authentication and Supervisord configuration
3. Startup sequencing and health checks
4. Debugging checklist
5. Helm installation and verification commands
6. File and configuration references
7. Next steps and recommendations

---

## 1. Deployment overview

This Helm chart deploys:

- A Deployment containing:
  - Init container to create directories and set permissions
  - `gowa` container: Admin API (Go binary) which manages instances and talks to Supervisord via XML-RPC
  - `supervisord` sidecar: supervisord daemon exposing an HTTP XML-RPC interface (port 9001)
- A Service (ClusterIP) exposing:
  - 8088 (Admin API)
  - 9001 (Supervisord RPC)
  - 3001–3010 (WhatsApp instance ports)
- A ConfigMap for `supervisord.conf`
- A PVC for persistent instance storage (default 10Gi)

Key volumes are mounted so both containers can access configs, logs and instances data.

---

## 2. Authentication and Supervisord configuration

Root cause for the 401 errors

- The Admin API expects environment variables named `SUPERVISOR_URL`, `SUPERVISOR_USER`, and `SUPERVISOR_PASS`.
- The supervisord ConfigMap must expose an HTTP XML-RPC interface on 127.0.0.1:9001 and use the same credentials.
- Mismatches (name format or different username/password) caused the Admin API to send empty or wrong credentials resulting in HTTP 401.

Working configuration (dev-style)

- Example `supervisord.conf` (dev):

```ini
[inet_http_server]
port=0.0.0.0:9001
username=admin
password=admin123

[supervisord]
logfile=/var/log/supervisor/supervisord.log
pidfile=/var/run/supervisord.pid
childlogdir=/var/log/supervisor
nodaemon=true
user=root
loglevel=debug

[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface

[supervisorctl]
serverurl=http://127.0.0.1:9001
username=admin
password=admin123

[include]
files = /etc/supervisor/conf.d/*.conf
```

- Corresponding environment variables in the Admin pod (values.yaml):

```yaml
SUPERVISOR_URL: "http://127.0.0.1:9001/RPC2"
SUPERVISOR_USER: "admin"
SUPERVISOR_PASS: "admin123"
```

Security note: In production, create a Kubernetes Secret for credentials instead of storing them in values.

---

## 3. Startup sequencing and health checks

Observed timing issues can cause the Admin API to attempt a connection before the supervisord HTTP endpoint is ready.

Implemented mitigations:

- Init container creates directories and fixes permissions before main containers start
- Admin container uses an inline `/bin/sh -c` startup script with a sleep delay (default 25s) to let supervisord initialize
- Probes are tuned:
  - Startup probe initialDelaySeconds: 30
  - Readiness probe initialDelaySeconds: 35
  - Liveness probe initialDelaySeconds: 40

Adjust these values if your environment needs longer startup time.

---

## 4. Debugging checklist

If you still see a 401 or connection problems, follow these steps in order (fastest first):

1. Inspect supervisord sidecar logs

```bash
kubectl logs deploy/<release-name> -c supervisord --tail=200
```

Look for lines like `INFO supervisord started with pid 1` and `INFO inet_http_server: started`.

2. Ensure the supervisord ConfigMap is mounted and contains the expected `[inet_http_server]` section

```bash
kubectl exec -it deploy/<release-name> -c supervisord -- cat /etc/supervisor/supervisord.conf
```

3. Verify Admin API environment variables inside the `gowa` container

```bash
kubectl exec -it deploy/<release-name> -c gowa -- env | grep SUPERVISOR
```

4. Manually test the RPC endpoint from inside the pod (use curl)

```bash
kubectl exec -it deploy/<release-name> -c gowa -- /bin/sh -c 'curl -u "$SUPERVISOR_USER:$SUPERVISOR_PASS" "$SUPERVISOR_URL" -v'
```

5. If supervisord isn't found in the image, check the sidecar image or install supervisord in the image used for that container.

6. If logs show `No file matches via include ".../conf.d/*.conf"`, it's just informational; add an empty `placeholder.conf` in the ConfigMap to silence the warning if desired.

7. If you still get 401, ensure there are no trailing spaces or different host values (use 127.0.0.1 instead of localhost) and that both server and client use the same credentials.

---

## 5. Helm installation and verification commands

Install or upgrade the chart:

```bash
helm upgrade --install gowa ./charts
```

Watch logs:

```bash
kubectl logs deployment/gowa -c supervisord -f
kubectl logs deployment/gowa -c gowa -f
```

Verify services and port forwarding:

```bash
kubectl get svc
kubectl port-forward svc/gowa 8088:8088 9001:9001
```

Health endpoints:

```bash
curl http://localhost:8088/healthz
curl http://localhost:8088/readyz
```

Manual RPC test:

```bash
curl -u admin:admin123 http://localhost:9001/RPC2
```

---

## 6. File and configuration references

- `charts/values.yaml` - chart default values (env vars, ports, volumes, probe timings)
- `charts/templates/deployment.yaml` - Deployment template with inline startup logic and supervisord sidecar
- `charts/templates/configmap.yaml` - supervisord.conf
- `.devcontainer/supervisord.conf` - reference working dev config

---

## 7. Next steps and recommendations

- Move Supervisor credentials to a Kubernetes Secret and update the chart to read from `valueFrom.secretKeyRef`.
- Add minimal Supervisor program templates into `/etc/supervisor/conf.d` via the ConfigMap to avoid the include warning and to define default instance programs.
- Consider keeping the Admin API and Supervisord in separate images if image size or dependency isolation is required.
- Tune probe timings to your environment, especially if running on slow nodes or with cold PVCs.

---

If you want, I can:
- Convert the supervisor username/password env vars to use a Secret and update the templates.
- Add a placeholder `conf.d/placeholder.conf` into the ConfigMap to silence the include warning.
          kubectl logs deployment/gowa -c supervisord -f

