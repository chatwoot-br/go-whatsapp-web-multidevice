# Admin API Guide

Complete guide for managing multiple WhatsApp instances using the Admin API with Supervisord.

## Table of Contents

- [Overview](#overview)
- [What is the Admin API](#what-is-the-admin-api)
- [Use Cases](#use-cases)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Usage Examples](#usage-examples)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Security](#security)

## Overview

The Admin API provides HTTP REST endpoints to dynamically create, manage, and delete GOWA (Go WhatsApp) instances. Each instance runs on a different port and is managed by Supervisord for robust process supervision.

**Key Features:**
- Create and delete instances programmatically
- Update instance configuration without stopping
- Multi-instance support for multiple WhatsApp accounts
- Automatic process supervision and restart
- Health monitoring and status checks
- Complete environment variable support
- Swagger UI documentation included

## What is the Admin API

The Admin API is a management layer that orchestrates multiple GOWA instances:

- **Programmatic Control**: Create/update/delete instances via HTTP REST API
- **Process Management**: Uses Supervisord for reliable process supervision
- **Configuration Management**: Generates and manages instance-specific configurations
- **Multi-Account Support**: Run multiple WhatsApp accounts on single server
- **Production Ready**: Includes health checks, logging, and error handling

**Architecture:**
```
Admin API (Port 8088)
    |
    +-- Supervisord (Process Manager)
            |
            +-- GOWA Instance 1 (Port 3001)
            +-- GOWA Instance 2 (Port 3002)
            +-- GOWA Instance 3 (Port 3003)
            +-- ...
```

## Use Cases

### 1. Multi-Tenant WhatsApp Service
Run separate WhatsApp instances for different clients or departments:
- Agency managing multiple client accounts
- Multi-brand customer support
- Department-specific communication

### 2. High Availability Setup
Deploy redundant instances with failover:
- Primary and backup instances
- Load distribution across accounts
- Geographic redundancy

### 3. Development and Testing
Create temporary instances for testing:
- Test new features without affecting production
- Automated testing environments
- Staging instances

### 4. Dynamic Scaling
Scale WhatsApp capacity on demand:
- Create instances during peak hours
- Remove instances when not needed
- Automatic provisioning based on load

## Prerequisites

### Required

- **Supervisord**: Process control system
  ```bash
  # Ubuntu/Debian
  sudo apt install supervisor

  # macOS
  brew install supervisor

  # Verify installation
  supervisorctl version
  ```

- **GOWA Binary**: WhatsApp application binary
  - Download from [releases](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases)
  - Or build from source

- **Directory Structure**:
  ```bash
  /etc/supervisor/conf.d/       # Supervisord config directory
  /app/instances/               # Instance data directory
  /var/log/supervisor/          # Log directory
  /tmp/                         # Lock directory
  /usr/local/bin/whatsapp       # GOWA binary location
  ```

### Optional

- **PostgreSQL**: For production database (recommended)
- **Reverse Proxy**: Nginx/Caddy for SSL termination
- **Monitoring**: Prometheus/Grafana for metrics

## Installation

### Option 1: Using Dev Container (Recommended for Development)

The project includes a VS Code Dev Container with everything pre-configured:

```bash
# 1. Clone repository
git clone https://github.com/chatwoot-br/go-whatsapp-web-multidevice
cd go-whatsapp-web-multidevice

# 2. Open in VS Code
code .

# 3. Reopen in Container (when prompted)
# Or: Ctrl+Shift+P -> "Dev Containers: Reopen in Container"

# 4. Use development helper
./.devcontainer/dev.sh start-admin
./.devcontainer/dev.sh create 3001
```

**Features included:**
- Go 1.24+ with all tools
- FFmpeg pre-installed
- Supervisord configured and running
- Admin API ready to use
- Port forwarding configured

### Option 2: Manual Installation

#### Step 1: Install Supervisord

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install supervisor

# macOS
brew install supervisor

# Start service
sudo systemctl start supervisor  # Linux
brew services start supervisor   # macOS
```

#### Step 2: Create Directory Structure

```bash
# Create required directories
sudo mkdir -p /etc/supervisor/conf.d
sudo mkdir -p /app/instances
sudo mkdir -p /var/log/supervisor
sudo mkdir -p /tmp

# Set permissions
sudo chmod 755 /etc/supervisor/conf.d
sudo chmod 755 /app/instances
sudo chmod 755 /var/log/supervisor
```

#### Step 3: Install GOWA Binary

```bash
# Download latest release
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-linux-amd64

# Install to system path
sudo mv whatsapp-linux-amd64 /usr/local/bin/whatsapp
sudo chmod +x /usr/local/bin/whatsapp

# Verify installation
whatsapp --version
```

#### Step 4: Configure Supervisord

Edit `/etc/supervisor/supervisord.conf` to enable XML-RPC interface:

```ini
[inet_http_server]
port = 127.0.0.1:9001
username = admin
password = admin

[supervisorctl]
serverurl = http://127.0.0.1:9001
username = admin
password = admin

[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface
```

Restart Supervisord:
```bash
sudo systemctl restart supervisor
```

## Configuration

### Environment Variables

#### Required

- `ADMIN_TOKEN`: Bearer token for API authentication
  ```bash
  export ADMIN_TOKEN="your-secure-token-here"
  ```

#### Supervisord Connection

- `SUPERVISOR_URL`: Supervisord XML-RPC endpoint (default: `http://127.0.0.1:9001/RPC2`)
- `SUPERVISOR_USER`: Supervisord username (default: `admin`)
- `SUPERVISOR_PASS`: Supervisord password (default: `admin`)

#### Paths

- `SUPERVISOR_CONF_DIR`: Config directory (default: `/etc/supervisor/conf.d/`)
- `INSTANCES_DIR`: Instance data directory (default: `/app/instances/`)
- `SUPERVISOR_LOG_DIR`: Log directory (default: `/var/log/supervisor/`)
- `LOCK_DIR`: Lock file directory (default: `/tmp/`)
- `GOWA_BIN`: GOWA binary path (default: `/usr/local/bin/whatsapp`)

#### Instance Defaults (Optional)

Set default configuration for all new instances:

- `GOWA_DEBUG`: Enable debug logging (default: `false`)
- `GOWA_OS`: Device name (default: `Chrome`)
- `GOWA_BASIC_AUTH`: Basic authentication credentials
- `GOWA_BASE_PATH`: Base path for subpath deployment
- `GOWA_AUTO_REPLY`: Auto-reply message
- `GOWA_AUTO_MARK_READ`: Auto-mark messages as read (default: `false`)
- `GOWA_WEBHOOK`: Default webhook URL
- `GOWA_WEBHOOK_SECRET`: Default webhook secret
- `GOWA_ACCOUNT_VALIDATION`: Validate accounts (default: `true`)
- `GOWA_CHAT_STORAGE`: Enable chat storage (default: `true`)

### Complete Configuration Example

```bash
# Admin API Configuration
export ADMIN_TOKEN="$(openssl rand -hex 32)"
export ADMIN_PORT=8088

# Supervisord Configuration
export SUPERVISOR_URL="http://127.0.0.1:9001/RPC2"
export SUPERVISOR_USER="admin"
export SUPERVISOR_PASS="admin"

# Paths
export SUPERVISOR_CONF_DIR="/etc/supervisor/conf.d/"
export INSTANCES_DIR="/app/instances/"
export SUPERVISOR_LOG_DIR="/var/log/supervisor/"
export GOWA_BIN="/usr/local/bin/whatsapp"

# Default Instance Settings (Optional)
export GOWA_DEBUG=false
export GOWA_OS="Production"
export GOWA_WEBHOOK="https://api.yourapp.com/webhooks/whatsapp"
export GOWA_WEBHOOK_SECRET="$(openssl rand -hex 32)"
```

### Start Admin API

```bash
# With environment variables set
./whatsapp admin --port 8088

# Or specify configuration via flags
./whatsapp admin \
  --port 8088 \
  --supervisor-url "http://127.0.0.1:9001/RPC2"
```

## API Reference

All endpoints require Bearer token authentication:
```
Authorization: Bearer your-secure-token-here
```

### Create Instance

**POST** `/admin/instances`

Create a new GOWA instance on the specified port.

**Request Body:**
```json
{
  "port": 3001,
  "basic_auth": "user:password",
  "debug": true,
  "os": "MyApp",
  "account_validation": false,
  "base_path": "/api",
  "auto_reply": "Auto reply message",
  "auto_mark_read": true,
  "webhook": "https://webhook.site/xxx",
  "webhook_secret": "super-secret",
  "chat_storage": true
}
```

**Minimal Request (only port required):**
```json
{
  "port": 3001
}
```

**Field Descriptions:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `port` | integer | Yes | - | Port number (1024-65535) |
| `basic_auth` | string | No | - | Basic auth (user:password) |
| `debug` | boolean | No | false | Enable debug logging |
| `os` | string | No | Chrome | Device name in WhatsApp |
| `account_validation` | boolean | No | true | Validate WhatsApp accounts |
| `base_path` | string | No | - | Base path for subpath deployment |
| `auto_reply` | string | No | - | Auto-reply message |
| `auto_mark_read` | boolean | No | false | Auto-mark messages as read |
| `webhook` | string | No | - | Webhook URL |
| `webhook_secret` | string | No | secret | Webhook HMAC secret |
| `chat_storage` | boolean | No | true | Enable chat history storage |

**Response (201 Created):**
```json
{
  "data": {
    "port": 3001,
    "state": "RUNNING",
    "pid": 12345,
    "uptime": "5s",
    "logs": {
      "stdout": "/var/log/supervisor/gowa_3001.out.log",
      "stderr": "/var/log/supervisor/gowa_3001.err.log"
    }
  },
  "message": "Instance created successfully",
  "request_id": "uuid-here",
  "timestamp": "2025-11-14T10:00:00Z"
}
```

**Error Responses:**
- `400` - Invalid port or JSON
- `409` - Instance already exists or port in use
- `500` - Internal server error

### List Instances

**GET** `/admin/instances`

List all GOWA instances with their current status.

**Response (200 OK):**
```json
{
  "data": [
    {
      "port": 3001,
      "state": "RUNNING",
      "pid": 12345,
      "uptime": "1h23m45s",
      "logs": {
        "stdout": "/var/log/supervisor/gowa_3001.out.log",
        "stderr": "/var/log/supervisor/gowa_3001.err.log"
      }
    },
    {
      "port": 3002,
      "state": "STOPPED",
      "pid": 0,
      "uptime": "0s",
      "logs": {
        "stdout": "/var/log/supervisor/gowa_3002.out.log",
        "stderr": "/var/log/supervisor/gowa_3002.err.log"
      }
    }
  ],
  "message": "Instances retrieved successfully",
  "request_id": "uuid-here",
  "timestamp": "2025-11-14T10:00:00Z"
}
```

### Get Instance

**GET** `/admin/instances/{port}`

Get detailed information about a specific instance.

**Response (200 OK):**
```json
{
  "data": {
    "port": 3001,
    "state": "RUNNING",
    "pid": 12345,
    "uptime": "1h23m45s",
    "logs": {
      "stdout": "/var/log/supervisor/gowa_3001.out.log",
      "stderr": "/var/log/supervisor/gowa_3001.err.log"
    }
  },
  "message": "Instance retrieved successfully",
  "request_id": "uuid-here",
  "timestamp": "2025-11-14T10:00:00Z"
}
```

**Error Responses:**
- `400` - Invalid port parameter
- `404` - Instance not found

### Update Instance

**PATCH** `/admin/instances/{port}`

Update configuration of an existing instance. Only provided fields will be updated (partial update supported).

**Request Body:**
```json
{
  "basic_auth": "newuser:newpassword",
  "debug": false,
  "os": "UpdatedApp",
  "webhook": "https://new-webhook.example.com/whatsapp"
}
```

**Response (200 OK):**
```json
{
  "data": {
    "port": 3001,
    "state": "RUNNING",
    "pid": 12346,
    "uptime": "3s",
    "logs": {
      "stdout": "/var/log/supervisor/gowa_3001.out.log",
      "stderr": "/var/log/supervisor/gowa_3001.err.log"
    }
  },
  "message": "Instance configuration updated successfully",
  "request_id": "uuid-here",
  "timestamp": "2025-11-14T10:00:00Z"
}
```

**Update Process:**
1. Acquires per-port lock
2. Stops the instance
3. Updates configuration atomically
4. Calls Supervisord `Update()` to reconcile
5. Starts the instance
6. Waits for RUNNING state

**Error Responses:**
- `400` - Invalid port parameter or JSON
- `404` - Instance not found
- `409` - Port locked by another operation
- `500` - Update failed

### Delete Instance

**DELETE** `/admin/instances/{port}`

Stop and delete an instance, removing all configuration and resources.

**Response (200 OK):**
```json
{
  "message": "Instance deleted successfully",
  "request_id": "uuid-here",
  "timestamp": "2025-11-14T10:00:00Z"
}
```

**Error Responses:**
- `400` - Invalid port parameter
- `404` - Instance not found
- `500` - Deletion failed

### Health Check

**GET** `/healthz`

Check the health status of the admin service.

**Response (200 OK):**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-14T10:00:00Z",
  "supervisor_healthy": true,
  "version": "1.0.0"
}
```

### Readiness Check

**GET** `/readyz`

Check if the service is ready to accept requests.

**Response (200 OK):**
```json
{
  "message": "Service is ready",
  "request_id": "uuid-here",
  "timestamp": "2025-11-14T10:00:00Z"
}
```

### Instance States

| State | Description |
|-------|-------------|
| `RUNNING` | Instance is running normally |
| `STOPPED` | Instance is stopped |
| `STARTING` | Instance is in the process of starting |
| `FATAL` | Instance failed to start or crashed |
| `UNKNOWN` | State could not be determined |

## Usage Examples

### Using curl

**Create instance (minimal):**
```bash
curl -X POST "http://localhost:8088/admin/instances" \
  -H "Authorization: Bearer your-secure-token-here" \
  -H "Content-Type: application/json" \
  -d '{"port": 3001}'
```

**Create instance with full configuration:**
```bash
curl -X POST "http://localhost:8088/admin/instances" \
  -H "Authorization: Bearer your-secure-token-here" \
  -H "Content-Type: application/json" \
  -d '{
    "port": 3001,
    "basic_auth": "admin:password123",
    "debug": true,
    "os": "MyCustomApp",
    "account_validation": false,
    "base_path": "/api/v1",
    "auto_reply": "This is an automated response",
    "auto_mark_read": true,
    "webhook": "https://webhook.example.com/whatsapp",
    "webhook_secret": "my-webhook-secret",
    "chat_storage": true
  }'
```

**List all instances:**
```bash
curl -X GET "http://localhost:8088/admin/instances" \
  -H "Authorization: Bearer your-secure-token-here"
```

**Get specific instance:**
```bash
curl -X GET "http://localhost:8088/admin/instances/3001" \
  -H "Authorization: Bearer your-secure-token-here"
```

**Update instance configuration:**
```bash
curl -X PATCH "http://localhost:8088/admin/instances/3001" \
  -H "Authorization: Bearer your-secure-token-here" \
  -H "Content-Type: application/json" \
  -d '{
    "basic_auth": "newuser:newpassword",
    "debug": false,
    "webhook": "https://new-webhook.example.com/whatsapp"
  }'
```

**Delete instance:**
```bash
curl -X DELETE "http://localhost:8088/admin/instances/3001" \
  -H "Authorization: Bearer your-secure-token-here"
```

**Health check:**
```bash
curl -X GET "http://localhost:8088/healthz"
```

### Using httpie

**Create instance:**
```bash
http POST localhost:8088/admin/instances \
  Authorization:"Bearer your-secure-token-here" \
  port:=3001 \
  basic_auth="admin:password123" \
  debug:=true
```

**List instances:**
```bash
http GET localhost:8088/admin/instances \
  Authorization:"Bearer your-secure-token-here"
```

**Update instance:**
```bash
http PATCH localhost:8088/admin/instances/3001 \
  Authorization:"Bearer your-secure-token-here" \
  debug:=false \
  webhook="https://new-webhook.example.com/whatsapp"
```

### Using Python

```python
import requests
import json

API_URL = "http://localhost:8088"
TOKEN = "your-secure-token-here"

headers = {
    "Authorization": f"Bearer {TOKEN}",
    "Content-Type": "application/json"
}

# Create instance
response = requests.post(
    f"{API_URL}/admin/instances",
    headers=headers,
    json={
        "port": 3001,
        "basic_auth": "admin:password123",
        "debug": True,
        "webhook": "https://webhook.example.com/whatsapp"
    }
)
print(response.json())

# List instances
response = requests.get(
    f"{API_URL}/admin/instances",
    headers=headers
)
print(response.json())

# Update instance
response = requests.patch(
    f"{API_URL}/admin/instances/3001",
    headers=headers,
    json={
        "debug": False,
        "webhook": "https://new-webhook.example.com/whatsapp"
    }
)
print(response.json())

# Delete instance
response = requests.delete(
    f"{API_URL}/admin/instances/3001",
    headers=headers
)
print(response.json())
```

### Using Node.js

```javascript
const axios = require('axios');

const API_URL = 'http://localhost:8088';
const TOKEN = 'your-secure-token-here';

const headers = {
  'Authorization': `Bearer ${TOKEN}`,
  'Content-Type': 'application/json'
};

// Create instance
async function createInstance() {
  const response = await axios.post(
    `${API_URL}/admin/instances`,
    {
      port: 3001,
      basic_auth: 'admin:password123',
      debug: true,
      webhook: 'https://webhook.example.com/whatsapp'
    },
    { headers }
  );
  console.log(response.data);
}

// List instances
async function listInstances() {
  const response = await axios.get(
    `${API_URL}/admin/instances`,
    { headers }
  );
  console.log(response.data);
}

// Update instance
async function updateInstance(port) {
  const response = await axios.patch(
    `${API_URL}/admin/instances/${port}`,
    {
      debug: false,
      webhook: 'https://new-webhook.example.com/whatsapp'
    },
    { headers }
  );
  console.log(response.data);
}

// Delete instance
async function deleteInstance(port) {
  const response = await axios.delete(
    `${API_URL}/admin/instances/${port}`,
    { headers }
  );
  console.log(response.data);
}
```

## Development

### Using Development Helper Script

The Dev Container includes a helper script for common operations:

```bash
# Start admin server
./.devcontainer/dev.sh start-admin

# Create instance
./.devcontainer/dev.sh create 3001

# List instances
./.devcontainer/dev.sh list

# Update instance
./.devcontainer/dev.sh update 3001 '{"debug": false, "webhook": "https://new-webhook.com"}'

# Delete instance
./.devcontainer/dev.sh delete 3001

# Show help
./.devcontainer/dev.sh help
```

### Example Development Workflow

```bash
# 1. Start admin server
./.devcontainer/dev.sh start-admin

# 2. Create instance with basic config
./.devcontainer/dev.sh create 3001

# 3. Update with webhook
./.devcontainer/dev.sh update 3001 '{"webhook": "https://webhook.site/unique-id", "debug": true}'

# 4. Check status
./.devcontainer/dev.sh list

# 5. Update webhook secret
./.devcontainer/dev.sh update 3001 '{"webhook_secret": "my-secret-key"}'

# 6. Delete when done
./.devcontainer/dev.sh delete 3001
```

### Generated Configuration Files

Each instance gets its own Supervisord configuration at:
`/etc/supervisor/conf.d/gowa-{port}.conf`

**Example:**
```ini
[program:gowa_3001]
command=/usr/local/bin/whatsapp rest --port=3001 --debug=false --os=Chrome --account-validation=false --basic-auth=admin:admin --auto-mark-read=true --webhook="https://webhook.site/xxx" --webhook-secret="super-secret-key"
directory=/app
autostart=true
autorestart=true
startretries=3
stdout_logfile=/var/log/supervisor/gowa_3001.out.log
stderr_logfile=/var/log/supervisor/gowa_3001.err.log
environment=APP_PORT="3001",APP_DEBUG="false",APP_OS="Chrome",APP_BASIC_AUTH="admin:admin",DB_URI="file:/app/instances/3001/storages/whatsapp.db?_foreign_keys=on",WHATSAPP_AUTO_MARK_READ="true",WHATSAPP_WEBHOOK="https://webhook.site/xxx",WHATSAPP_WEBHOOK_SECRET="super-secret-key",WHATSAPP_ACCOUNT_VALIDATION="false",WHATSAPP_CHAT_STORAGE="true"
```

## Troubleshooting

### Common Issues

**1. "ADMIN_TOKEN environment variable is required"**

**Solution:** Set the required environment variable:
```bash
export ADMIN_TOKEN="your-secure-token-here"
```

**2. "Failed to connect to supervisord"**

**Causes:**
- Supervisord not running
- Wrong SUPERVISOR_URL
- Authentication credentials incorrect

**Solution:**
```bash
# Check if Supervisord is running
sudo systemctl status supervisor  # Linux
brew services list | grep supervisor  # macOS

# Test connection manually
supervisorctl status

# Restart Supervisord
sudo systemctl restart supervisor  # Linux
brew services restart supervisor  # macOS

# Check configuration
cat /etc/supervisor/supervisord.conf | grep inet_http_server -A 3
```

**3. "Port validation failed"**

**Solution:** Use valid port range (1024-65535):
```bash
# Valid
curl -X POST .../admin/instances -d '{"port": 3001}'

# Invalid
curl -X POST .../admin/instances -d '{"port": 80}'    # Too low
curl -X POST .../admin/instances -d '{"port": 70000}' # Too high
```

**4. "Port is currently locked by another operation"**

**Cause:** Another admin operation is in progress for that port

**Solution:**
```bash
# Wait for operation to complete (usually < 30s)
# Or check for stale lock files
ls -la /tmp/gowa.*.lock

# Remove stale locks (be careful!)
rm /tmp/gowa.3001.lock
```

**5. "Instance creation failed"**

**Causes:**
- Binary not found
- Permission issues
- Port already in use

**Solution:**
```bash
# Check binary exists
ls -la /usr/local/bin/whatsapp
which whatsapp

# Check permissions
sudo chmod +x /usr/local/bin/whatsapp

# Check port availability
netstat -tulpn | grep 3001
lsof -i :3001

# Check Supervisord logs
tail -f /var/log/supervisor/supervisord.log
tail -f /var/log/supervisor/gowa_3001.err.log
```

### Log Files

**Admin server logs:**
- Console output (stdout/stderr)
- Configure log forwarding as needed

**Instance logs:**
- Standard output: `/var/log/supervisor/gowa_{port}.out.log`
- Standard error: `/var/log/supervisor/gowa_{port}.err.log`

**Supervisord logs:**
- Main log: `/var/log/supervisor/supervisord.log`

**View logs:**
```bash
# Admin server (if running via systemd)
sudo journalctl -u whatsapp-admin -f

# Instance logs
tail -f /var/log/supervisor/gowa_3001.out.log
tail -f /var/log/supervisor/gowa_3001.err.log

# Supervisord logs
tail -f /var/log/supervisor/supervisord.log
```

### Lock Files

Lock files prevent concurrent modifications to the same port:

**Location:** `/tmp/gowa.{port}.lock`

**Check locks:**
```bash
ls -la /tmp/gowa.*.lock
```

**Remove stale locks:**
```bash
# Only if operation is truly stuck
rm /tmp/gowa.3001.lock
```

## Security

### Authentication

**Bearer Token Authentication:**
- All protected endpoints require `Authorization: Bearer <token>` header
- Token must be set via `ADMIN_TOKEN` environment variable
- Use strong, randomly generated tokens (minimum 32 characters)

**Generate secure token:**
```bash
openssl rand -hex 32
```

**Best Practices:**
- Rotate tokens regularly (every 90 days)
- Store tokens in secrets manager (AWS Secrets Manager, HashiCorp Vault)
- Never commit tokens to version control
- Use different tokens per environment (dev, staging, prod)
- Monitor failed authentication attempts

### Network Security

**1. Bind to localhost only:**
```bash
# Only accessible from local machine
./whatsapp admin --port 8088  # Binds to 127.0.0.1 by default
```

**2. Use reverse proxy for external access:**
```nginx
server {
    listen 443 ssl http2;
    server_name admin.yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:8088;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;

        # Rate limiting
        limit_req zone=admin burst=10 nodelay;

        # IP whitelist (optional)
        allow 192.168.1.0/24;
        deny all;
    }
}
```

**3. Firewall rules:**
```bash
# Allow only specific IPs
sudo ufw allow from 192.168.1.0/24 to any port 8088

# Or keep port closed and use reverse proxy
sudo ufw deny 8088
```

### Supervisord Security

**1. Never expose XML-RPC publicly:**
```ini
# Good: Bind to localhost only
[inet_http_server]
port = 127.0.0.1:9001

# Bad: Accessible from anywhere
[inet_http_server]
port = 0.0.0.0:9001
```

**2. Use authentication:**
```ini
[inet_http_server]
port = 127.0.0.1:9001
username = admin
password = strong-password-here
```

**3. File permissions:**
```bash
# Restrict config directory
sudo chmod 755 /etc/supervisor/conf.d
sudo chown root:root /etc/supervisor/conf.d

# Restrict log directory
sudo chmod 755 /var/log/supervisor
sudo chown root:root /var/log/supervisor
```

### Instance Security

Each instance inherits all GOWA security features:

- **Basic Authentication**: Set via `basic_auth` field
- **Webhook HMAC**: Set via `webhook_secret` field
- **Database Isolation**: Separate database per instance
- **Process Isolation**: Separate process per instance

**Security Checklist:**

- [ ] Strong ADMIN_TOKEN configured
- [ ] Admin API bound to localhost only
- [ ] Supervisord not exposed publicly
- [ ] Supervisord authentication enabled
- [ ] File permissions properly set
- [ ] TLS/HTTPS via reverse proxy
- [ ] Rate limiting configured
- [ ] Monitoring and alerting enabled
- [ ] Log files secured
- [ ] Regular security updates

## Kubernetes/Helm Deployment

For production Kubernetes deployments:

```bash
# Install with Swagger UI enabled
helm install my-release charts/gowa --set swaggerUI.enabled=true

# Access Swagger UI (after port forwarding)
kubectl port-forward svc/my-release-gowa 8080:8080
open http://localhost:8080/swagger
```

See `charts/gowa/README.md` for complete Helm chart documentation.

## Additional Resources

- **OpenAPI Specification**: `docs/admin-api-openapi.yaml` - Complete API specification
- **Implementation Details**: `docs/features/ADR-001/IMPLEMENTATION_SUMMARY.md` - Technical architecture
- **Swagger UI Guide**: `docs/SWAGGER-UI-INTEGRATION-COMPLETE.md` - Interactive API documentation
- **Dev Container Guide**: `.devcontainer/README.md` - Development environment setup
- **GitHub Repository**: [go-whatsapp-web-multidevice](https://github.com/chatwoot-br/go-whatsapp-web-multidevice)

---

**Version**: Compatible with v7.7.0+
**Last Updated**: 2025-11-14
