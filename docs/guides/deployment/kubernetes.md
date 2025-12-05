# Kubernetes Deployment Guide

This guide covers deploying the WhatsApp Web API Multidevice application on Kubernetes clusters.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Deployment Methods](#deployment-methods)
- [Basic Kubernetes Deployment](#basic-kubernetes-deployment)
- [Helm Chart Deployment](#helm-chart-deployment)
- [Production Kubernetes Setup](#production-kubernetes-setup)
- [Scaling Considerations](#scaling-considerations)
- [Monitoring and Troubleshooting](#monitoring-and-troubleshooting)
- [Related Guides](#related-guides)

## Prerequisites

### Required

- **Kubernetes Cluster**: Version 1.20 or higher
- **kubectl**: Kubernetes command-line tool configured to access your cluster
- **Persistent Storage**: Storage class available in your cluster (for data persistence)
- **WhatsApp Account**: Active WhatsApp account with phone

### Optional

- **Helm**: Version 3.0 or higher (for Helm chart deployment)
- **Ingress Controller**: For external access (nginx-ingress, Traefik, etc.)
- **Cert-Manager**: For automatic SSL certificate management
- **Monitoring Tools**: Prometheus, Grafana for monitoring

### Installing Tools

**kubectl:**
```bash
# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# macOS
brew install kubectl

# Verify installation
kubectl version --client
```

**Helm:**
```bash
# Linux/macOS
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# macOS (alternative)
brew install helm

# Verify installation
helm version
```

## Deployment Methods

### Method 1: Raw Kubernetes Manifests

Deploy using standard Kubernetes YAML files (covered in this guide).

### Method 2: Helm Chart

Use the official Helm chart from the repository (recommended for production):

```bash
# Add Helm repository (when available)
helm repo add gowa https://chatwoot-br.github.io/go-whatsapp-web-multidevice

# Or use local chart
cd charts/gowa
helm install whatsapp-api .
```

## Basic Kubernetes Deployment

### Create Namespace

```bash
kubectl create namespace whatsapp
```

### Create Secrets

Create `secrets.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: whatsapp-secret
  namespace: whatsapp
type: Opaque
stringData:
  basic-auth: "admin:secret123"
  webhook-secret: "your-webhook-secret"
```

Apply:
```bash
kubectl apply -f secrets.yaml
```

### Create ConfigMap

Create `configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: whatsapp-config
  namespace: whatsapp
data:
  webhook-url: "https://your-webhook.com/handler"
  app-port: "3000"
  app-debug: "false"
  app-os: "MyAppName"
```

Apply:
```bash
kubectl apply -f configmap.yaml
```

### Create Persistent Volume Claim

Create `pvc.yaml`:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: whatsapp-pvc
  namespace: whatsapp
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  # Specify storage class if needed
  # storageClassName: standard
```

Apply:
```bash
kubectl apply -f pvc.yaml
```

### Create Deployment

Create `deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: whatsapp-api
  namespace: whatsapp
  labels:
    app: whatsapp
spec:
  replicas: 1
  selector:
    matchLabels:
      app: whatsapp
  template:
    metadata:
      labels:
        app: whatsapp
    spec:
      containers:
      - name: whatsapp
        image: ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest
        ports:
        - containerPort: 3000
          name: http
        env:
        - name: APP_PORT
          valueFrom:
            configMapKeyRef:
              name: whatsapp-config
              key: app-port
        - name: APP_DEBUG
          valueFrom:
            configMapKeyRef:
              name: whatsapp-config
              key: app-debug
        - name: APP_OS
          valueFrom:
            configMapKeyRef:
              name: whatsapp-config
              key: app-os
        - name: APP_BASIC_AUTH
          valueFrom:
            secretKeyRef:
              name: whatsapp-secret
              key: basic-auth
        - name: WHATSAPP_WEBHOOK
          valueFrom:
            configMapKeyRef:
              name: whatsapp-config
              key: webhook-url
        - name: WHATSAPP_WEBHOOK_SECRET
          valueFrom:
            secretKeyRef:
              name: whatsapp-secret
              key: webhook-secret
        volumeMounts:
        - name: storage
          mountPath: /app/storages
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /app/devices
            port: 3000
            httpHeaders:
            - name: Authorization
              value: Basic YWRtaW46c2VjcmV0MTIz  # base64 of admin:secret123
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 5
        readinessProbe:
          httpGet:
            path: /app/devices
            port: 3000
            httpHeaders:
            - name: Authorization
              value: Basic YWRtaW46c2VjcmV0MTIz
          initialDelaySeconds: 10
          periodSeconds: 10
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: whatsapp-pvc
```

Apply:
```bash
kubectl apply -f deployment.yaml
```

### Create Service

Create `service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: whatsapp-service
  namespace: whatsapp
  labels:
    app: whatsapp
spec:
  type: ClusterIP
  selector:
    app: whatsapp
  ports:
  - port: 80
    targetPort: 3000
    protocol: TCP
    name: http
```

For external access via LoadBalancer:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: whatsapp-service
  namespace: whatsapp
  labels:
    app: whatsapp
spec:
  type: LoadBalancer
  selector:
    app: whatsapp
  ports:
  - port: 80
    targetPort: 3000
    protocol: TCP
    name: http
```

Apply:
```bash
kubectl apply -f service.yaml
```

### Verify Deployment

```bash
# Check all resources
kubectl get all -n whatsapp

# Check pod logs
kubectl logs -f deployment/whatsapp-api -n whatsapp

# Check service
kubectl get svc -n whatsapp

# Port forward for testing
kubectl port-forward -n whatsapp svc/whatsapp-service 3000:80
```

Access the application:
- Web UI: `http://localhost:3000`

## Helm Chart Deployment

### Using Official Helm Chart

The repository includes a Helm chart in `charts/gowa/`.

**Install from local chart:**

```bash
# Navigate to chart directory
cd charts/gowa

# Review values
cat values.yaml

# Install chart
helm install whatsapp-api . \
  --namespace whatsapp \
  --create-namespace

# Or with custom values
helm install whatsapp-api . \
  --namespace whatsapp \
  --create-namespace \
  --set basicAuth.username=admin \
  --set basicAuth.password=secret123 \
  --set webhook.url=https://your-webhook.com/handler \
  --set webhook.secret=your-secret
```

**Custom values.yaml:**

Create `my-values.yaml`:

```yaml
# Application settings
replicaCount: 1

image:
  repository: ghcr.io/chatwoot-br/go-whatsapp-web-multidevice
  tag: "v7.8.3"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 80

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: whatsapp.yourdomain.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: whatsapp-tls
      hosts:
        - whatsapp.yourdomain.com

# Basic Authentication
basicAuth:
  enabled: true
  username: admin
  password: secret123

# Webhook configuration
webhook:
  url: "https://your-webhook.com/handler"
  secret: "your-webhook-secret"

# Persistence
persistence:
  enabled: true
  storageClass: ""
  size: 5Gi

# Resources
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"

# Auto-reply (optional)
autoReply:
  enabled: false
  message: "Thanks for your message!"

# WhatsApp settings
whatsapp:
  autoMarkRead: false
  accountValidation: true
  chatStorage: true
```

Install with custom values:

```bash
helm install whatsapp-api ./charts/gowa \
  --namespace whatsapp \
  --create-namespace \
  -f my-values.yaml
```

### Helm Management

```bash
# List releases
helm list -n whatsapp

# Upgrade release
helm upgrade whatsapp-api ./charts/gowa \
  --namespace whatsapp \
  -f my-values.yaml

# Rollback release
helm rollback whatsapp-api 1 -n whatsapp

# Uninstall release
helm uninstall whatsapp-api -n whatsapp

# View release values
helm get values whatsapp-api -n whatsapp
```

## Production Kubernetes Setup

### Ingress with SSL

Create `ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: whatsapp-ingress
  namespace: whatsapp
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "300"
spec:
  tls:
  - hosts:
    - whatsapp.yourdomain.com
    secretName: whatsapp-tls
  rules:
  - host: whatsapp.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: whatsapp-service
            port:
              number: 80
```

Apply:
```bash
kubectl apply -f ingress.yaml
```

### PostgreSQL Database

For production, use PostgreSQL instead of SQLite.

**Using external PostgreSQL:**

Update deployment environment variables:

```yaml
env:
- name: DB_URI
  valueFrom:
    secretKeyRef:
      name: whatsapp-secret
      key: db-uri
```

Add to secrets:

```yaml
stringData:
  db-uri: "postgresql://whatsapp:password@postgres-service:5432/whatsapp?sslmode=disable"
```

**Using PostgreSQL in cluster:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: whatsapp
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        env:
        - name: POSTGRES_DB
          value: whatsapp
        - name: POSTGRES_USER
          value: whatsapp
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: whatsapp-secret
              key: db-password
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        ports:
        - containerPort: 5432
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
  namespace: whatsapp
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
```

### Resource Quotas

Create `resourcequota.yaml`:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: whatsapp-quota
  namespace: whatsapp
spec:
  hard:
    requests.cpu: "2"
    requests.memory: 2Gi
    limits.cpu: "4"
    limits.memory: 4Gi
    persistentvolumeclaims: "3"
```

### Network Policies

Create `networkpolicy.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: whatsapp-netpol
  namespace: whatsapp
spec:
  podSelector:
    matchLabels:
      app: whatsapp
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 3000
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 80
```

## Scaling Considerations

### Important Limitations

⚠️ **Warning**: The application cannot run multiple replicas with the same WhatsApp account due to WhatsApp's protocol limitations.

**For single WhatsApp account:**
- Keep `replicas: 1` in deployment
- Use single persistent volume
- Cannot scale horizontally with same account

**For multiple WhatsApp accounts:**
- Deploy separate instances per account
- Use different namespaces or names
- Maintain separate databases and volumes

### Multi-Account Setup

Deploy multiple instances for different accounts:

```bash
# Account 1
helm install whatsapp-account1 ./charts/gowa \
  --namespace whatsapp-acc1 \
  --create-namespace \
  --set basicAuth.username=admin1 \
  --set basicAuth.password=pass1

# Account 2
helm install whatsapp-account2 ./charts/gowa \
  --namespace whatsapp-acc2 \
  --create-namespace \
  --set basicAuth.username=admin2 \
  --set basicAuth.password=pass2
```

### Vertical Scaling

Adjust resources based on usage:

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

Update deployment:
```bash
kubectl apply -f deployment.yaml
```

## Monitoring and Troubleshooting

### View Logs

```bash
# View pod logs
kubectl logs -f deployment/whatsapp-api -n whatsapp

# View logs from specific pod
kubectl logs -f whatsapp-api-xxxxx-yyyyy -n whatsapp

# View previous logs (if pod crashed)
kubectl logs --previous whatsapp-api-xxxxx-yyyyy -n whatsapp

# Stream logs with timestamps
kubectl logs -f deployment/whatsapp-api -n whatsapp --timestamps
```

### Check Pod Status

```bash
# Get pod details
kubectl get pods -n whatsapp
kubectl describe pod whatsapp-api-xxxxx-yyyyy -n whatsapp

# Check pod events
kubectl get events -n whatsapp --sort-by='.lastTimestamp'

# Check resource usage
kubectl top pod -n whatsapp
kubectl top node
```

### Debug Pod Issues

```bash
# Execute shell in pod
kubectl exec -it deployment/whatsapp-api -n whatsapp -- sh

# Check environment variables
kubectl exec deployment/whatsapp-api -n whatsapp -- env

# Check mounted volumes
kubectl exec deployment/whatsapp-api -n whatsapp -- ls -la /app/storages

# Test connectivity
kubectl exec deployment/whatsapp-api -n whatsapp -- wget -O- http://localhost:3000/app/devices
```

### Port Forward for Testing

```bash
# Forward service port
kubectl port-forward -n whatsapp svc/whatsapp-service 3000:80

# Forward pod port directly
kubectl port-forward -n whatsapp pod/whatsapp-api-xxxxx-yyyyy 3000:3000
```

Access at: `http://localhost:3000`

### Common Issues

**Pod in CrashLoopBackOff:**

```bash
# Check logs
kubectl logs deployment/whatsapp-api -n whatsapp

# Check pod events
kubectl describe pod whatsapp-api-xxxxx-yyyyy -n whatsapp

# Common causes:
# - Invalid configuration
# - Missing secrets/configmaps
# - Volume mount issues
# - Resource limits too low
```

**PVC Pending:**

```bash
# Check PVC status
kubectl get pvc -n whatsapp
kubectl describe pvc whatsapp-pvc -n whatsapp

# Check available storage classes
kubectl get storageclass

# Check PV binding
kubectl get pv
```

**ImagePullBackOff:**

```bash
# Check image pull status
kubectl describe pod whatsapp-api-xxxxx-yyyyy -n whatsapp

# Verify image exists
docker pull ghcr.io/chatwoot-br/go-whatsapp-web-multidevice:latest

# Check image pull secrets if using private registry
kubectl get secrets -n whatsapp
```

### Health Checks

Test application health:

```bash
# Using kubectl
kubectl exec deployment/whatsapp-api -n whatsapp -- \
  wget -qO- http://localhost:3000/app/devices

# Using port-forward
kubectl port-forward -n whatsapp svc/whatsapp-service 3000:80 &
curl -u admin:secret123 http://localhost:3000/app/devices
```

## Backup and Restore

### Backup Persistent Volume

```bash
# Create backup pod
kubectl run backup --image=alpine --restart=Never -n whatsapp \
  --overrides='
{
  "spec": {
    "containers": [{
      "name": "backup",
      "image": "alpine",
      "command": ["tar", "czf", "/backup/whatsapp-backup.tar.gz", "-C", "/data", "."],
      "volumeMounts": [
        {"name": "storage", "mountPath": "/data"},
        {"name": "backup", "mountPath": "/backup"}
      ]
    }],
    "volumes": [
      {"name": "storage", "persistentVolumeClaim": {"claimName": "whatsapp-pvc"}},
      {"name": "backup", "hostPath": {"path": "/tmp/backup"}}
    ]
  }
}'

# Copy backup from pod
kubectl cp whatsapp/backup:/backup/whatsapp-backup.tar.gz ./whatsapp-backup.tar.gz

# Clean up
kubectl delete pod backup -n whatsapp
```

### Restore from Backup

```bash
# Upload backup
kubectl cp ./whatsapp-backup.tar.gz whatsapp/whatsapp-api-xxxxx-yyyyy:/tmp/

# Extract in pod
kubectl exec -it deployment/whatsapp-api -n whatsapp -- \
  tar xzf /tmp/whatsapp-backup.tar.gz -C /app/storages

# Restart pod
kubectl rollout restart deployment/whatsapp-api -n whatsapp
```

## Related Guides

- **[Docker Deployment Guide](docker.md)** - Deploy using Docker and Docker Compose
- **[Binary Deployment Guide](binary.md)** - Deploy using pre-built binaries
- **[Production Checklist](production-checklist.md)** - Production deployment best practices
- **[Main Deployment Guide](../../deployment-guide.md)** - Overview of all deployment methods

## Additional Resources

- **Official Helm Chart**: `charts/gowa/` in repository
- **API Documentation**: `docs/openapi.yaml` - Full REST API specification
- **Webhook Guide**: `docs/webhook-payload.md` - Webhook integration
- **GitHub Repository**: [go-whatsapp-web-multidevice](https://github.com/chatwoot-br/go-whatsapp-web-multidevice)

---

**Version**: Compatible with v7.7.0+
**Last Updated**: 2025-10-05
