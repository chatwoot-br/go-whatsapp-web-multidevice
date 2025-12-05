# Configuration Reference

Complete reference of all configuration options for the WhatsApp Web API Multidevice application.

## Table of Contents

- [Configuration Methods](#configuration-methods)
- [Application Settings](#application-settings)
- [Database Configuration](#database-configuration)
- [WhatsApp Features](#whatsapp-features)
- [Webhook Configuration](#webhook-configuration)
- [Security Settings](#security-settings)
- [Media Settings](#media-settings)
- [Performance Settings](#performance-settings)
- [Admin API Configuration](#admin-api-configuration)
- [Environment Files](#environment-files)
- [Examples](#examples)

## Configuration Methods

The application supports three configuration methods in order of priority (highest to lowest):

### 1. Command-Line Flags (Highest Priority)

Override any configuration with explicit command-line arguments:

```bash
./whatsapp rest --port 8080 --debug=true -b admin:secret
```

**View all available flags:**
```bash
./whatsapp rest --help
./whatsapp mcp --help
./whatsapp admin --help
```

### 2. Environment Variables

Set configuration via environment variables (system-wide or shell session):

```bash
export APP_PORT=8080
export APP_DEBUG=true
export APP_BASIC_AUTH=admin:secret
./whatsapp rest
```

**Good for:**
- Docker and containerized deployments
- System-wide configuration
- Orchestration tools (Docker Compose, Kubernetes)
- CI/CD pipelines

### 3. .env File (Lowest Priority)

Create `src/.env` file for persistent configuration:

```bash
APP_PORT=8080
APP_DEBUG=true
APP_BASIC_AUTH=admin:secret
```

**Good for:**
- Development environments
- Multiple configuration profiles
- Version control (with .gitignore)
- Quick testing and prototyping

**Create from example:**
```bash
cp src/.env.example src/.env
# Edit src/.env with your values
```

## Application Settings

### APP_PORT

HTTP server port for REST API mode.

| Property | Value |
|----------|-------|
| **Environment Variable** | `APP_PORT` |
| **CLI Flag** | `--port` |
| **Default** | `3000` |
| **Type** | Integer |
| **Range** | 1024-65535 |

**Examples:**
```bash
# Environment variable
export APP_PORT=8080

# CLI flag
./whatsapp rest --port 8080

# .env file
APP_PORT=8080
```

**Use Cases:**
- Avoid port conflicts with other services
- Run multiple instances on different ports
- Standard ports with reverse proxy (80, 443)
- Container port mapping

---

### APP_DEBUG

Enable detailed debug logging for troubleshooting.

| Property | Value |
|----------|-------|
| **Environment Variable** | `APP_DEBUG` |
| **CLI Flag** | `--debug` |
| **Default** | `false` |
| **Type** | Boolean |

**Examples:**
```bash
# Environment variable
export APP_DEBUG=true

# CLI flag
./whatsapp rest --debug=true

# .env file
APP_DEBUG=true
```

**Debug Output Includes:**
- WhatsApp protocol messages
- Connection events
- Message sending/receiving details
- Database queries
- API request/response details

**Production:** Set to `false` to reduce log verbosity and improve performance.

---

### APP_OS

Device name displayed in WhatsApp linked devices.

| Property | Value |
|----------|-------|
| **Environment Variable** | `APP_OS` |
| **CLI Flag** | `--os` |
| **Default** | `Chrome` |
| **Type** | String |

**Examples:**
```bash
# Environment variable
export APP_OS="MyApp"

# CLI flag
./whatsapp rest --os="Production API"

# .env file
APP_OS=Customer Support Bot
```

**Use Cases:**
- Identify different instances
- Branding and descriptive names
- Differentiate dev/staging/prod
- Multi-tenant identification

**Appears as:** "Chrome (MyApp)" or your custom value in WhatsApp mobile app's linked devices.

---

### APP_BASIC_AUTH

Basic HTTP authentication credentials for API access.

| Property | Value |
|----------|-------|
| **Environment Variable** | `APP_BASIC_AUTH` |
| **CLI Flag** | `-b, --basic-auth` |
| **Default** | None (disabled) |
| **Type** | String |
| **Format** | `username:password` or `user1:pass1,user2:pass2` |

**Examples:**
```bash
# Single user
export APP_BASIC_AUTH="admin:secret123"

# Multiple users (comma-separated)
export APP_BASIC_AUTH="admin:secret123,api:apikey456,support:support789"

# CLI flag
./whatsapp rest -b admin:secret123

# .env file
APP_BASIC_AUTH=admin:secret123,api:apikey456
```

**Security Best Practices:**
- Use passwords with at least 20 characters
- Generate passwords: `openssl rand -base64 32`
- Different credentials per user
- Rotate credentials regularly (every 90 days)
- Store in secrets manager (AWS Secrets Manager, HashiCorp Vault)
- Always use with HTTPS in production

---

### APP_BASE_PATH

Base URL path for subpath deployment (reverse proxy scenarios).

| Property | Value |
|----------|-------|
| **Environment Variable** | `APP_BASE_PATH` |
| **CLI Flag** | `--base-path` |
| **Default** | `/` (root) |
| **Type** | String |
| **Format** | Path starting with `/` |

**Examples:**
```bash
# Environment variable
export APP_BASE_PATH=/whatsapp-api

# CLI flag
./whatsapp rest --base-path="/whatsapp-api"

# .env file
APP_BASE_PATH=/api/v1/whatsapp
```

**Access URLs:**
- Web UI: `http://localhost:3000/whatsapp-api/`
- Send message: `http://localhost:3000/whatsapp-api/send/message`
- Login: `http://localhost:3000/whatsapp-api/app/login`

**Nginx Configuration Example:**
```nginx
location /whatsapp-api/ {
    proxy_pass http://localhost:3000/whatsapp-api/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

---

### APP_TRUSTED_PROXY

Enable trusted proxy mode for deployments behind reverse proxies.

| Property | Value |
|----------|-------|
| **Environment Variable** | `APP_TRUSTED_PROXY` |
| **CLI Flag** | `--trusted-proxy` |
| **Default** | `false` |
| **Type** | Boolean |

**Examples:**
```bash
# Environment variable
export APP_TRUSTED_PROXY=true

# CLI flag
./whatsapp rest --trusted-proxy=true

# .env file
APP_TRUSTED_PROXY=true
```

**When Enabled:**
- Trusts `X-Forwarded-For` header for real client IP
- Trusts `X-Forwarded-Proto` for original protocol
- Trusts `X-Real-IP` header

**Use When:**
- Behind nginx, Caddy, Traefik
- Behind load balancer
- Using Cloudflare or similar CDN
- Need accurate client IP logging

## Database Configuration

### DB_URI

Main database connection string for WhatsApp session data.

| Property | Value |
|----------|-------|
| **Environment Variable** | `DB_URI` |
| **CLI Flag** | `--db-uri` |
| **Default** | `file:storages/whatsapp.db?_foreign_keys=on` |
| **Type** | String (URI) |

**SQLite Examples:**
```bash
# Default SQLite
export DB_URI="file:storages/whatsapp.db?_foreign_keys=on"

# SQLite with WAL mode (better performance)
export DB_URI="file:storages/whatsapp.db?_journal_mode=WAL&_timeout=5000&_foreign_keys=on"

# In-memory (testing only)
export DB_URI="file::memory:?cache=shared"
```

**PostgreSQL Examples:**
```bash
# PostgreSQL (recommended for production)
export DB_URI="postgresql://whatsapp:password@localhost:5432/whatsapp?sslmode=disable"

# PostgreSQL with SSL
export DB_URI="postgresql://whatsapp:password@localhost:5432/whatsapp?sslmode=require"

# PostgreSQL with connection pool
export DB_URI="postgresql://whatsapp:password@localhost:5432/whatsapp?pool_max_conns=10"
```

**PostgreSQL Setup:**
```sql
CREATE DATABASE whatsapp;
CREATE USER whatsapp WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE whatsapp TO whatsapp;
ALTER DATABASE whatsapp OWNER TO whatsapp;
```

**Comparison:**

| Feature | SQLite | PostgreSQL |
|---------|--------|------------|
| Setup | None required | Requires PostgreSQL server |
| Performance (Single) | Excellent | Good |
| Performance (Multi) | Limited | Excellent |
| Concurrent Writers | 1 | Many |
| Scaling | Vertical only | Horizontal + Vertical |
| Backup | File copy | pg_dump |
| Best For | Single instance | Production, multiple instances |

---

### DB_KEYS_URI

Encryption keys database connection string.

| Property | Value |
|----------|-------|
| **Environment Variable** | `DB_KEYS_URI` |
| **CLI Flag** | `--db-keys-uri` |
| **Default** | `file::memory:?cache=shared&_foreign_keys=on` |
| **Type** | String (URI) |

**Examples:**
```bash
# In-memory (default, fastest)
export DB_KEYS_URI="file::memory:?cache=shared&_foreign_keys=on"

# Persistent file (keys preserved across restarts)
export DB_KEYS_URI="file:storages/keys.db?_foreign_keys=on"
```

**In-Memory (Default):**
- Faster performance
- No disk I/O
- Keys regenerated on restart
- Good for most use cases

**Persistent:**
- Keys preserved across restarts
- Required for some advanced features
- Slight performance overhead

## WhatsApp Features

### WHATSAPP_AUTO_REPLY

Automatically reply to incoming messages with predefined text.

| Property | Value |
|----------|-------|
| **Environment Variable** | `WHATSAPP_AUTO_REPLY` |
| **CLI Flag** | `--autoreply` |
| **Default** | None (disabled) |
| **Type** | String |

**Examples:**
```bash
# Environment variable
export WHATSAPP_AUTO_REPLY="Thanks for your message! We'll reply soon."

# CLI flag
./whatsapp rest --autoreply="Thank you for contacting us!"

# .env file
WHATSAPP_AUTO_REPLY=We received your message and will respond within 2 hours.
```

**Use Cases:**
- Out-of-office messages
- Acknowledgment messages
- Automated responses
- Business hours notifications

**Behavior:**
- Replies to every incoming message
- Works with both individual and group messages
- Sends immediately upon message receipt

---

### WHATSAPP_AUTO_MARK_READ

Automatically mark incoming messages as read.

| Property | Value |
|----------|-------|
| **Environment Variable** | `WHATSAPP_AUTO_MARK_READ` |
| **CLI Flag** | `--auto-mark-read` |
| **Default** | `false` |
| **Type** | Boolean |

**Examples:**
```bash
# Environment variable
export WHATSAPP_AUTO_MARK_READ=true

# CLI flag
./whatsapp rest --auto-mark-read=true

# .env file
WHATSAPP_AUTO_MARK_READ=true
```

**When Enabled:**
- All incoming messages marked as read immediately
- Sender sees blue checkmarks
- No "unread" indicator in WhatsApp

**Use Cases:**
- Automation scenarios
- Prevent unread message buildup
- Hide processing status from senders

---

### WHATSAPP_ACCOUNT_VALIDATION

Validate phone numbers exist on WhatsApp before sending messages.

| Property | Value |
|----------|-------|
| **Environment Variable** | `WHATSAPP_ACCOUNT_VALIDATION` |
| **CLI Flag** | `--account-validation` |
| **Default** | `true` |
| **Type** | Boolean |

**Examples:**
```bash
# Environment variable
export WHATSAPP_ACCOUNT_VALIDATION=true

# CLI flag
./whatsapp rest --account-validation=true

# .env file
WHATSAPP_ACCOUNT_VALIDATION=false
```

**When Enabled (true):**
- Checks if phone number is registered on WhatsApp
- Returns error if number doesn't exist
- Prevents wasted API calls
- Slight delay on first send to new number

**When Disabled (false):**
- No validation check
- Faster message sending
- May send to invalid numbers

**Recommendation:** Keep enabled (`true`) for production to avoid errors.

---

### WHATSAPP_CHAT_STORAGE

Store chat history in local database for querying.

| Property | Value |
|----------|-------|
| **Environment Variable** | `WHATSAPP_CHAT_STORAGE` |
| **CLI Flag** | `--chat-storage` |
| **Default** | `true` |
| **Type** | Boolean |

**Examples:**
```bash
# Environment variable
export WHATSAPP_CHAT_STORAGE=true

# CLI flag
./whatsapp rest --chat-storage=false

# .env file
WHATSAPP_CHAT_STORAGE=true
```

**When Enabled (true):**
- All messages stored in `storages/chatstorage.db`
- Enables chat history API endpoints
- Allows message queries and search
- Increased disk usage
- Slight performance overhead

**When Disabled (false):**
- No message history stored
- Reduced disk usage
- Better performance
- Privacy-focused
- Chat API endpoints return empty

**Storage Location:** `storages/chatstorage.db` (SQLite)

## Webhook Configuration

### WHATSAPP_WEBHOOK

Webhook URL(s) to receive real-time WhatsApp events.

| Property | Value |
|----------|-------|
| **Environment Variable** | `WHATSAPP_WEBHOOK` |
| **CLI Flag** | `-w, --webhook` |
| **Default** | None (disabled) |
| **Type** | String (URL or comma-separated URLs) |

**Examples:**
```bash
# Single webhook
export WHATSAPP_WEBHOOK="https://your-webhook.site/handler"

# Multiple webhooks (comma-separated)
export WHATSAPP_WEBHOOK="https://webhook1.com/handler,https://webhook2.com/handler"

# CLI flag
./whatsapp rest -w "https://webhook.example.com/whatsapp"

# .env file
WHATSAPP_WEBHOOK=https://api.yourapp.com/webhooks/whatsapp,https://backup-webhook.com/handler
```

**Webhook Events:**
- New messages received
- Message delivery receipts
- Message read receipts
- Group events (joins, leaves, updates)
- Connection status changes
- Protocol events

**Requirements:**
- Must be publicly accessible HTTPS URL
- Must return `200 OK` status
- Should respond within 5 seconds
- Should verify HMAC signature

**See:** [Webhook Payload Documentation](webhooks/payload-schemas.md) for complete event reference.

---

### WHATSAPP_WEBHOOK_SECRET

HMAC secret for webhook signature verification.

| Property | Value |
|----------|-------|
| **Environment Variable** | `WHATSAPP_WEBHOOK_SECRET` |
| **CLI Flag** | `--webhook-secret` |
| **Default** | `secret` |
| **Type** | String |
| **Recommended Length** | 32+ characters |

**Examples:**
```bash
# Generate strong secret
WEBHOOK_SECRET=$(openssl rand -hex 32)

# Environment variable
export WHATSAPP_WEBHOOK_SECRET="$WEBHOOK_SECRET"

# CLI flag
./whatsapp rest --webhook-secret="super-secret-key-abc123"

# .env file
WHATSAPP_WEBHOOK_SECRET=a3d8f7e2b9c1d4e5f6a7b8c9d0e1f2a3
```

**Security:**
- Generate random secret: `openssl rand -hex 32`
- Minimum 32 characters recommended
- Change default value in production
- Store in secrets manager
- Rotate periodically (every 90 days)

**Signature Verification:**
- HMAC SHA256 algorithm
- Sent in `X-Hub-Signature-256` header
- Format: `sha256={hash}`

**See:** [Webhook Security Guide](../guides/webhooks/security.md) for verification examples.

## Security Settings

### Security Environment Variables Summary

| Variable | Default | Purpose |
|----------|---------|---------|
| `APP_BASIC_AUTH` | None | HTTP Basic Authentication |
| `WHATSAPP_WEBHOOK_SECRET` | `secret` | Webhook HMAC signature |
| `APP_TRUSTED_PROXY` | `false` | Trust proxy headers |
| `ADMIN_TOKEN` | - | Admin API bearer token |

**Security Checklist:**

- [ ] Strong basic auth credentials set
- [ ] Webhook secret changed from default
- [ ] HTTPS enabled for webhooks
- [ ] Reverse proxy with SSL configured
- [ ] Firewall rules configured
- [ ] Rate limiting enabled
- [ ] Regular security updates
- [ ] Monitoring and alerting enabled

**See:** [Security Best Practices Guide](../operations/security-best-practices.md) for comprehensive security configuration.

## Media Settings

### Media Size Limits

Configure maximum file sizes for media processing.

| Variable | Default | Description |
|----------|---------|-------------|
| `WHATSAPP_SETTING_MAX_IMAGE_SIZE` | 20971520 (20MB) | Maximum image file size |
| `WHATSAPP_SETTING_MAX_VIDEO_SIZE` | 104857600 (100MB) | Maximum video file size |
| `WHATSAPP_SETTING_MAX_FILE_SIZE` | 52428800 (50MB) | Maximum document file size |
| `WHATSAPP_SETTING_MAX_AUDIO_SIZE` | 16777216 (16MB) | Maximum audio file size |
| `WHATSAPP_SETTING_MAX_DOWNLOAD_SIZE` | 524288000 (500MB) | Maximum download size |

**Examples:**
```bash
# Environment variables (bytes)
export WHATSAPP_SETTING_MAX_IMAGE_SIZE=20971520    # 20MB
export WHATSAPP_SETTING_MAX_VIDEO_SIZE=104857600   # 100MB
export WHATSAPP_SETTING_MAX_FILE_SIZE=52428800     # 50MB
export WHATSAPP_SETTING_MAX_AUDIO_SIZE=16777216    # 16MB

# .env file
WHATSAPP_SETTING_MAX_IMAGE_SIZE=20971520
WHATSAPP_SETTING_MAX_VIDEO_SIZE=104857600
WHATSAPP_SETTING_MAX_FILE_SIZE=52428800
WHATSAPP_SETTING_MAX_AUDIO_SIZE=16777216
```

**WhatsApp Limits:**
- Images: 20MB recommended
- Videos: 100MB maximum
- Audio: 16MB maximum (WhatsApp limit)
- Documents: 50MB recommended

**FFmpeg Required:**
Media processing requires FFmpeg to be installed on the system.

**See:** [Media Handling Guide](../guides/media-handling.md) for detailed information.

---

### WHATSAPP_SETTING_AUTO_CONVERT_AUDIO

Automatically convert audio files to optimal WhatsApp format.

| Property | Value |
|----------|-------|
| **Environment Variable** | `WHATSAPP_SETTING_AUTO_CONVERT_AUDIO` |
| **Default** | `true` |
| **Type** | Boolean |

**Examples:**
```bash
# Environment variable
export WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=true

# .env file
WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=true
```

**When Enabled:**
- Converts audio to WhatsApp-compatible format (Opus in OGG container)
- Optimizes file size
- Ensures compatibility
- Requires FFmpeg

## Performance Settings

### Performance Optimization Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `WHATSAPP_CHAT_STORAGE` | `true` | Enable/disable chat history |
| `DB_URI` | SQLite | Use PostgreSQL for better performance |
| `DB_KEYS_URI` | In-memory | Keys database location |

**High Performance Configuration:**
```bash
# Use PostgreSQL instead of SQLite
export DB_URI="postgresql://whatsapp:pass@localhost:5432/whatsapp"

# Disable chat storage if not needed
export WHATSAPP_CHAT_STORAGE=false

# Keep keys in memory
export DB_KEYS_URI="file::memory:?cache=shared"

# Disable debug logging
export APP_DEBUG=false
```

**Resource Usage:**
- Memory: ~256MB base + 50-100MB per connection
- CPU: Low (1-2% idle, spikes during media processing)
- Disk: Varies by chat storage and media caching

## Admin API Configuration

Configuration for the Admin API (multi-instance management).

### ADMIN_TOKEN

Bearer token for Admin API authentication (REQUIRED).

| Property | Value |
|----------|-------|
| **Environment Variable** | `ADMIN_TOKEN` |
| **CLI Flag** | - |
| **Default** | None (required) |
| **Type** | String |
| **Recommended Length** | 32+ characters |

**Examples:**
```bash
# Generate secure token
export ADMIN_TOKEN="$(openssl rand -hex 32)"

# Or set manually
export ADMIN_TOKEN="your-secure-token-here"
```

---

### ADMIN_PORT

HTTP port for Admin API server.

| Property | Value |
|----------|-------|
| **Environment Variable** | `ADMIN_PORT` |
| **CLI Flag** | `--port` |
| **Default** | `8088` |
| **Type** | Integer |

---

### Supervisord Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SUPERVISOR_URL` | `http://127.0.0.1:9001/RPC2` | Supervisord XML-RPC endpoint |
| `SUPERVISOR_USER` | `admin` | Supervisord username |
| `SUPERVISOR_PASS` | `admin` | Supervisord password |
| `SUPERVISOR_CONF_DIR` | `/etc/supervisor/conf.d/` | Config directory |
| `INSTANCES_DIR` | `/app/instances/` | Instance data directory |
| `SUPERVISOR_LOG_DIR` | `/var/log/supervisor/` | Log directory |
| `LOCK_DIR` | `/tmp/` | Lock file directory |
| `GOWA_BIN` | `/usr/local/bin/whatsapp` | GOWA binary path |

**Example:**
```bash
export ADMIN_TOKEN="$(openssl rand -hex 32)"
export ADMIN_PORT=8088
export SUPERVISOR_URL="http://127.0.0.1:9001/RPC2"
export SUPERVISOR_USER="admin"
export SUPERVISOR_PASS="admin"
```

### Instance Defaults (GOWA_* Variables)

Set default configuration for all instances created via Admin API:

| Variable | Default | Maps To |
|----------|---------|---------|
| `GOWA_DEBUG` | `false` | `APP_DEBUG` |
| `GOWA_OS` | `Chrome` | `APP_OS` |
| `GOWA_BASIC_AUTH` | - | `APP_BASIC_AUTH` |
| `GOWA_BASE_PATH` | - | `APP_BASE_PATH` |
| `GOWA_AUTO_REPLY` | - | `WHATSAPP_AUTO_REPLY` |
| `GOWA_AUTO_MARK_READ` | `false` | `WHATSAPP_AUTO_MARK_READ` |
| `GOWA_WEBHOOK` | - | `WHATSAPP_WEBHOOK` |
| `GOWA_WEBHOOK_SECRET` | `secret` | `WHATSAPP_WEBHOOK_SECRET` |
| `GOWA_ACCOUNT_VALIDATION` | `true` | `WHATSAPP_ACCOUNT_VALIDATION` |
| `GOWA_CHAT_STORAGE` | `true` | `WHATSAPP_CHAT_STORAGE` |

**Example:**
```bash
# Set defaults for all new instances
export GOWA_DEBUG=false
export GOWA_OS="Production API"
export GOWA_WEBHOOK="https://api.yourapp.com/webhooks/whatsapp"
export GOWA_WEBHOOK_SECRET="$(openssl rand -hex 32)"
```

**See:** [Admin API Guide](../guides/admin-api.md) for complete documentation.

## Environment Files

### .env File Format

Create `src/.env` file with configuration:

```bash
# Application Settings
APP_PORT=3000
APP_DEBUG=false
APP_OS=Chrome
APP_BASIC_AUTH=admin:password123,user:password456
APP_BASE_PATH=

# Database Settings
DB_URI=file:storages/whatsapp.db?_foreign_keys=on
DB_KEYS_URI=file::memory:?cache=shared&_foreign_keys=on

# WhatsApp Features
WHATSAPP_AUTO_REPLY=Thanks for your message!
WHATSAPP_AUTO_MARK_READ=false
WHATSAPP_ACCOUNT_VALIDATION=true
WHATSAPP_CHAT_STORAGE=true

# Webhook Settings
WHATSAPP_WEBHOOK=https://webhook.site/your-id,https://backup-webhook.com/handler
WHATSAPP_WEBHOOK_SECRET=super-secret-key-xyz

# Media Settings
WHATSAPP_SETTING_MAX_IMAGE_SIZE=20971520
WHATSAPP_SETTING_MAX_FILE_SIZE=52428800
WHATSAPP_SETTING_MAX_VIDEO_SIZE=104857600
WHATSAPP_SETTING_MAX_AUDIO_SIZE=16777216
WHATSAPP_SETTING_AUTO_CONVERT_AUDIO=true

# Timezone (for Docker)
TZ=America/Sao_Paulo
```

### Docker Compose Environment

```yaml
version: '3.8'

services:
  whatsapp:
    image: ghcr.io/chatwoot-br/go-whatsapp-web-multidevice
    container_name: whatsapp
    restart: always
    ports:
      - "3000:3000"
    volumes:
      - whatsapp-data:/app/storages
    environment:
      # Application
      - APP_PORT=3000
      - APP_DEBUG=false
      - APP_OS=Production
      - APP_BASIC_AUTH=admin:${ADMIN_PASSWORD}
      - APP_TRUSTED_PROXY=true

      # Database
      - DB_URI=postgresql://whatsapp:${DB_PASSWORD}@postgres:5432/whatsapp

      # WhatsApp
      - WHATSAPP_AUTO_REPLY=Thanks for contacting us!
      - WHATSAPP_AUTO_MARK_READ=false
      - WHATSAPP_ACCOUNT_VALIDATION=true
      - WHATSAPP_CHAT_STORAGE=true

      # Webhook
      - WHATSAPP_WEBHOOK=${WEBHOOK_URL}
      - WHATSAPP_WEBHOOK_SECRET=${WEBHOOK_SECRET}

      # Timezone
      - TZ=America/Sao_Paulo
    command:
      - rest

volumes:
  whatsapp-data:
```

### Systemd Environment File

Create `/etc/whatsapp/config.env`:

```bash
APP_PORT=3000
APP_DEBUG=false
APP_OS=Production
APP_BASIC_AUTH=admin:secret123
WHATSAPP_WEBHOOK=https://webhook.site/your-id
WHATSAPP_WEBHOOK_SECRET=super-secret
```

Reference in systemd service file:
```ini
[Service]
EnvironmentFile=/etc/whatsapp/config.env
```

## Examples

### Development Configuration

```bash
# Quick local development
./whatsapp rest \
  --port 3000 \
  --debug=true \
  --os="Dev" \
  --chat-storage=true
```

### Production Configuration

```bash
# Secure production deployment
./whatsapp rest \
  --port 3000 \
  --debug=false \
  --os="Production API" \
  -b "admin:$(openssl rand -base64 32)" \
  --db-uri="postgresql://whatsapp:password@localhost:5432/whatsapp" \
  --webhook="https://api.yourapp.com/webhooks/whatsapp" \
  --webhook-secret="$(openssl rand -hex 32)" \
  --account-validation=true \
  --chat-storage=true \
  --trusted-proxy=true
```

### Customer Support Configuration

```bash
# With auto-reply and webhooks
./whatsapp rest \
  --port 3000 \
  --os="Support Bot" \
  -b "support:secret123" \
  --autoreply="Thanks for contacting support! A team member will respond within 2 hours." \
  --auto-mark-read=true \
  --webhook="https://support.yourapp.com/webhooks/whatsapp" \
  --webhook-secret="your-secret"
```

### Multi-Instance Configuration

```bash
# Instance 1 (Account 1)
./whatsapp rest \
  --port 3001 \
  --os="Account 1" \
  -b "account1:pass1" \
  --db-uri="file:storages/account1.db" \
  --webhook="https://webhook.site/account1"

# Instance 2 (Account 2)
./whatsapp rest \
  --port 3002 \
  --os="Account 2" \
  -b "account2:pass2" \
  --db-uri="file:storages/account2.db" \
  --webhook="https://webhook.site/account2"
```

### High Performance Configuration

```bash
# Optimized for performance
./whatsapp rest \
  --port 3000 \
  --debug=false \
  --db-uri="postgresql://whatsapp:pass@localhost:5432/whatsapp" \
  --chat-storage=false \
  --account-validation=false
```

### Testing Configuration

```bash
# For testing and development
./whatsapp rest \
  --port 8080 \
  --debug=true \
  --os="Test Instance" \
  --account-validation=false \
  --chat-storage=false
```

## Configuration Validation

### Check Current Configuration

```bash
# Check if application is running
curl -I http://localhost:3000

# Test authentication (if enabled)
curl -u admin:password http://localhost:3000/app/devices

# Check device name and info
curl http://localhost:3000/user/info

# Check connection status
curl http://localhost:3000/app/devices
```

### Common Configuration Mistakes

**1. Port Conflicts:**
```bash
# Check if port is in use
netstat -tulpn | grep 3000
lsof -i :3000

# Solution: Use different port
./whatsapp rest --port 8080
```

**2. Authentication Format:**
```bash
# ❌ Wrong
APP_BASIC_AUTH="admin:pass word"  # No spaces in password
APP_BASIC_AUTH="admin pass"        # Missing colon

# ✅ Correct
APP_BASIC_AUTH="admin:password"
APP_BASIC_AUTH="admin:pass-word_123"
```

**3. Webhook URL:**
```bash
# ❌ Wrong
WHATSAPP_WEBHOOK="webhook.site"           # Missing protocol
WHATSAPP_WEBHOOK="http://localhost:3000"  # Not publicly accessible

# ✅ Correct
WHATSAPP_WEBHOOK="https://webhook.site/your-id"
WHATSAPP_WEBHOOK="https://api.yourapp.com/webhook"
```

**4. Database URI:**
```bash
# ❌ Wrong
DB_URI="postgres://localhost/whatsapp"  # Wrong protocol
DB_URI="file:whatsapp.db"               # Missing path

# ✅ Correct
DB_URI="postgresql://user:pass@localhost:5432/whatsapp"
DB_URI="file:storages/whatsapp.db?_foreign_keys=on"
```

## Quick Reference Tables

### All Configuration Variables

| Category | Variable | Default | CLI Flag |
|----------|----------|---------|----------|
| **Application** | | | |
| | `APP_PORT` | `3000` | `--port` |
| | `APP_DEBUG` | `false` | `--debug` |
| | `APP_OS` | `Chrome` | `--os` |
| | `APP_BASIC_AUTH` | - | `-b, --basic-auth` |
| | `APP_BASE_PATH` | - | `--base-path` |
| | `APP_TRUSTED_PROXY` | `false` | `--trusted-proxy` |
| **Database** | | | |
| | `DB_URI` | `file:storages/whatsapp.db` | `--db-uri` |
| | `DB_KEYS_URI` | `file::memory:` | `--db-keys-uri` |
| **WhatsApp** | | | |
| | `WHATSAPP_AUTO_REPLY` | - | `--autoreply` |
| | `WHATSAPP_AUTO_MARK_READ` | `false` | `--auto-mark-read` |
| | `WHATSAPP_WEBHOOK` | - | `-w, --webhook` |
| | `WHATSAPP_WEBHOOK_SECRET` | `secret` | `--webhook-secret` |
| | `WHATSAPP_ACCOUNT_VALIDATION` | `true` | `--account-validation` |
| | `WHATSAPP_CHAT_STORAGE` | `true` | `--chat-storage` |
| **Media** | | | |
| | `WHATSAPP_SETTING_MAX_IMAGE_SIZE` | `20971520` | - |
| | `WHATSAPP_SETTING_MAX_VIDEO_SIZE` | `104857600` | - |
| | `WHATSAPP_SETTING_MAX_FILE_SIZE` | `52428800` | - |
| | `WHATSAPP_SETTING_MAX_AUDIO_SIZE` | `16777216` | - |
| | `WHATSAPP_SETTING_AUTO_CONVERT_AUDIO` | `true` | - |
| **Admin API** | | | |
| | `ADMIN_TOKEN` | - | - |
| | `ADMIN_PORT` | `8088` | `--port` |
| | `SUPERVISOR_URL` | `http://127.0.0.1:9001/RPC2` | - |
| | `SUPERVISOR_USER` | `admin` | - |
| | `SUPERVISOR_PASS` | `admin` | - |

### Port Defaults by Mode

| Mode | Default Port | CLI Flag |
|------|--------------|----------|
| REST API | 3000 | `--port` |
| MCP Server | 8080 | `--port` |
| Admin API | 8088 | `--port` |

## Related Documentation

- **[Getting Started: Configuration Basics](../getting-started/configuration-basics.md)** - Step-by-step configuration guide
- **[Admin API Guide](../guides/admin-api.md)** - Multi-instance management
- **[Security Best Practices](../operations/security-best-practices.md)** - Production security configuration
- **[Deployment Guides](../guides/deployment/)** - Platform-specific deployment
- **[Media Handling Guide](../guides/media-handling.md)** - Media processing configuration

---

**Version**: Compatible with v7.10.1+
**Last Updated**: 2025-12-05
