# Configuration Basics

Essential configuration options to customize your WhatsApp Web API installation.

## Table of Contents

- [Configuration Methods](#configuration-methods)
- [Essential Configuration Options](#essential-configuration-options)
- [Security Configuration](#security-configuration)
- [WhatsApp Feature Configuration](#whatsapp-feature-configuration)
- [Database Configuration](#database-configuration)
- [Advanced Configuration](#advanced-configuration)
- [Configuration Examples](#configuration-examples)

## Configuration Methods

The application supports three configuration methods (in order of priority):

### 1. Command-Line Flags (Highest Priority)

```bash
./whatsapp rest --port 8080 --debug=true -b admin:secret
```

**Advantages:**
- Override any other configuration
- Easy for testing and quick changes
- Clear and explicit

**View all available flags:**
```bash
./whatsapp rest --help
```

### 2. Environment Variables

```bash
export APP_PORT=8080
export APP_DEBUG=true
export APP_BASIC_AUTH=admin:secret
./whatsapp rest
```

**Advantages:**
- Good for Docker and containerized deployments
- System-wide configuration
- Works with orchestration tools (Docker Compose, Kubernetes)

### 3. .env File (Lowest Priority)

Create `src/.env` file:

```bash
APP_PORT=8080
APP_DEBUG=true
APP_BASIC_AUTH=admin:secret
```

**Advantages:**
- Easy to manage multiple configurations
- Git-friendly (add to .gitignore)
- Good for development

**Create from example:**
```bash
cp src/.env.example src/.env
# Edit src/.env with your values
```

## Essential Configuration Options

### Application Settings

#### Port Configuration

**Change HTTP server port:**

```bash
# Command-line flag
./whatsapp rest --port 8080

# Environment variable
export APP_PORT=8080

# .env file
APP_PORT=8080
```

**Default:** 3000

**Use case:** Avoid port conflicts or use standard ports (80, 443 with reverse proxy)

#### Debug Mode

**Enable detailed logging:**

```bash
# Command-line flag
./whatsapp rest --debug=true

# Environment variable
export APP_DEBUG=true

# .env file
APP_DEBUG=true
```

**Default:** false

**Use case:** Troubleshooting, development, viewing detailed WhatsApp protocol messages

**Example debug output:**
```
[DEBUG] 2025-11-14 10:30:00 | Connecting to WhatsApp servers...
[DEBUG] 2025-11-14 10:30:01 | Connected successfully
[DEBUG] 2025-11-14 10:30:05 | Sending message to 5511999998888@s.whatsapp.net
[DEBUG] 2025-11-14 10:30:06 | Message sent: 3EB0C431D4D2E2D2F3E8
```

#### Device Name

**Change device name shown in WhatsApp:**

```bash
# Command-line flag
./whatsapp rest --os="MyApp"

# Environment variable
export APP_OS="MyApp"

# .env file
APP_OS=MyApp
```

**Default:** Chrome

**Use case:** Identify different instances, branding, or descriptive names

**How it appears:** When you check linked devices in WhatsApp mobile app, you'll see "Chrome (MyApp)" or "MyApp" depending on the value.

#### Base Path (Subpath Deployment)

**Deploy under a specific path:**

```bash
# Command-line flag
./whatsapp rest --base-path="/whatsapp-api"

# Environment variable
export APP_BASE_PATH=/whatsapp-api

# .env file
APP_BASE_PATH=/whatsapp-api
```

**Default:** / (root)

**Access:**
- Web UI: `http://localhost:3000/whatsapp-api/`
- API: `http://localhost:3000/whatsapp-api/send/message`

**Use case:** Integrate with existing web services, reverse proxy setups

**Example nginx configuration:**
```nginx
location /whatsapp-api/ {
    proxy_pass http://localhost:3000/whatsapp-api/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

## Security Configuration

### Basic Authentication

**Protect your API with username and password:**

```bash
# Single user
./whatsapp rest -b admin:secret123

# Multiple users (comma-separated)
./whatsapp rest -b admin:secret123,user1:pass1,user2:pass2

# Environment variable
export APP_BASIC_AUTH=admin:secret123,user1:pass1

# .env file
APP_BASIC_AUTH=admin:secret123,user1:pass1,user2:pass2
```

**Access API with authentication:**

```bash
# Using curl -u flag
curl -u admin:secret123 http://localhost:3000/app/devices

# Using Authorization header
curl -H "Authorization: Basic $(echo -n admin:secret123 | base64)" \
  http://localhost:3000/app/devices

# In browser
# Browser will prompt for username and password
```

**Best practices:**
- Use strong passwords (at least 16 characters)
- Different credentials per user
- Change credentials regularly
- Use HTTPS in production
- Store credentials securely (environment variables, secrets manager)

**Example strong configuration:**
```bash
APP_BASIC_AUTH=admin:$(openssl rand -base64 32),api_user:$(openssl rand -base64 32)
```

### Trusted Proxy Settings

**For deployments behind reverse proxies:**

Configure your reverse proxy to pass real client IP:

**Nginx:**
```nginx
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
```

**Caddy:**
```
reverse_proxy localhost:3000 {
    header_up X-Real-IP {remote_host}
    header_up X-Forwarded-For {remote_host}
}
```

## WhatsApp Feature Configuration

### Auto-Reply

**Automatically reply to incoming messages:**

```bash
# Command-line flag
./whatsapp rest --autoreply="Thanks for your message! We'll reply soon."

# Environment variable
export WHATSAPP_AUTO_REPLY="Thanks for your message! We'll reply soon."

# .env file
WHATSAPP_AUTO_REPLY=Thanks for your message! We'll reply soon.
```

**Default:** None (disabled)

**Use case:** Out-of-office messages, acknowledgment messages, automated responses

**Notes:**
- Replies to every incoming message
- Sends only once per conversation (configurable)
- Works with both individual and group messages

### Auto-Mark Read

**Automatically mark incoming messages as read:**

```bash
# Command-line flag
./whatsapp rest --auto-mark-read=true

# Environment variable
export WHATSAPP_AUTO_MARK_READ=true

# .env file
WHATSAPP_AUTO_MARK_READ=true
```

**Default:** false

**Use case:** Prevent sender from seeing "unread" status, automation scenarios

### Webhook Configuration

**Receive real-time events from WhatsApp:**

```bash
# Single webhook
./whatsapp rest --webhook="https://your-webhook.site/handler"

# Multiple webhooks (comma-separated)
./whatsapp rest --webhook="https://webhook1.com/handler,https://webhook2.com/handler"

# Environment variable
export WHATSAPP_WEBHOOK=https://your-webhook.site/handler

# .env file
WHATSAPP_WEBHOOK=https://your-webhook.site/handler,https://webhook2.com/handler
```

**Default:** None (disabled)

**Webhook events:**
- New messages
- Message delivery receipts
- Message read receipts
- Group events (joins, leaves, updates)
- Connection status changes

**Example webhook payload:**
```json
{
  "event": "message.received",
  "data": {
    "message_id": "3EB0C431D4D2E2D2F3E8",
    "from": "5511999998888@s.whatsapp.net",
    "text": "Hello!",
    "timestamp": "2025-11-14T10:30:00Z"
  }
}
```

See [Webhook Documentation](../webhook-payload.md) for complete details.

### Webhook Secret

**Secure webhook with HMAC signature:**

```bash
# Command-line flag
./whatsapp rest \
  --webhook="https://your-webhook.site/handler" \
  --webhook-secret="super-secret-key-$(openssl rand -hex 16)"

# Environment variable
export WHATSAPP_WEBHOOK_SECRET=super-secret-key-abc123

# .env file
WHATSAPP_WEBHOOK_SECRET=super-secret-key-abc123
```

**Default:** secret

**Verification:**

The webhook includes an HMAC-SHA256 signature in the `X-Webhook-Signature` header.

**Verify signature (Node.js example):**
```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(JSON.stringify(payload));
  const calculated = hmac.digest('hex');
  return calculated === signature;
}

// In webhook handler
app.post('/handler', (req, res) => {
  const signature = req.headers['x-webhook-signature'];
  const isValid = verifyWebhook(req.body, signature, 'super-secret-key-abc123');

  if (!isValid) {
    return res.status(401).send('Invalid signature');
  }

  // Process webhook
  console.log('Valid webhook:', req.body);
  res.status(200).send('OK');
});
```

### Account Validation

**Validate phone numbers before sending:**

```bash
# Command-line flag
./whatsapp rest --account-validation=true

# Environment variable
export WHATSAPP_ACCOUNT_VALIDATION=true

# .env file
WHATSAPP_ACCOUNT_VALIDATION=true
```

**Default:** true

**When enabled:**
- Checks if phone number is registered on WhatsApp before sending
- Returns error if number doesn't exist
- Prevents wasted API calls

**When to disable:**
- Testing with fake numbers
- Speed is more important than validation
- You're certain all numbers are valid

### Chat Storage

**Store chat history in database:**

```bash
# Command-line flag
./whatsapp rest --chat-storage=true

# Environment variable
export WHATSAPP_CHAT_STORAGE=true

# .env file
WHATSAPP_CHAT_STORAGE=true
```

**Default:** true

**When enabled:**
- Stores all incoming and outgoing messages
- Enables chat history API endpoints
- Allows message queries and search
- Database file: `storages/chatstorage.db`

**When to disable:**
- Privacy concerns
- Limited disk space
- Don't need message history

## Database Configuration

### Main Database

**Store WhatsApp connection data:**

```bash
# SQLite (default)
./whatsapp rest --db-uri="file:storages/whatsapp.db?_foreign_keys=on"

# PostgreSQL
./whatsapp rest --db-uri="postgresql://user:password@localhost:5432/whatsapp?sslmode=disable"

# Environment variable
export DB_URI="postgresql://user:password@localhost:5432/whatsapp"

# .env file
DB_URI=postgresql://user:password@localhost:5432/whatsapp?sslmode=disable
```

**Default:** `file:storages/whatsapp.db?_foreign_keys=on`

**SQLite advantages:**
- No additional setup required
- Lightweight
- Good for single instance

**PostgreSQL advantages:**
- Better for multiple instances
- Better performance at scale
- Better for production

**PostgreSQL setup:**
```bash
# Install PostgreSQL
sudo apt install postgresql

# Create database
sudo -u postgres psql
CREATE DATABASE whatsapp;
CREATE USER whatsapp WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE whatsapp TO whatsapp;
\q

# Run with PostgreSQL
./whatsapp rest --db-uri="postgresql://whatsapp:your-password@localhost:5432/whatsapp"
```

### Keys Database

**Store encryption keys:**

```bash
# In-memory (default, fastest)
./whatsapp rest --db-keys-uri="file::memory:?cache=shared&_foreign_keys=on"

# SQLite file (persistent)
./whatsapp rest --db-keys-uri="file:storages/keys.db?_foreign_keys=on"

# Environment variable
export DB_KEYS_URI="file:storages/keys.db?_foreign_keys=on"

# .env file
DB_KEYS_URI=file:storages/keys.db?_foreign_keys=on
```

**Default:** `file::memory:?cache=shared&_foreign_keys=on`

**In-memory advantages:**
- Faster
- No disk I/O
- Keys regenerated on restart

**Persistent advantages:**
- Keys preserved across restarts
- Required for some advanced features

## Advanced Configuration

### Complete Configuration Example

**src/.env file with all options:**

```bash
# Application Settings
APP_PORT=3000
APP_DEBUG=false
APP_OS=MyWhatsAppAPI
APP_BASIC_AUTH=admin:strong-password-here,api_user:another-strong-password
APP_BASE_PATH=

# Database Settings
DB_URI=file:storages/whatsapp.db?_foreign_keys=on
DB_KEYS_URI=file::memory:?cache=shared&_foreign_keys=on

# WhatsApp Features
WHATSAPP_AUTO_REPLY=Thanks for contacting us! We'll respond soon.
WHATSAPP_AUTO_MARK_READ=true
WHATSAPP_ACCOUNT_VALIDATION=true
WHATSAPP_CHAT_STORAGE=true

# Webhook Settings
WHATSAPP_WEBHOOK=https://webhook.site/your-unique-id,https://backup-webhook.com/handler
WHATSAPP_WEBHOOK_SECRET=super-secret-signing-key-xyz

# Timezone (for Docker)
TZ=America/Sao_Paulo
```

### Docker Configuration

**Using command-line flags:**

```bash
docker run -d \
  --name whatsapp \
  --publish 3000:3000 \
  --volume whatsapp-data:/app/storages \
  --restart always \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest \
  --port 3000 \
  --debug false \
  --os "MyApp" \
  -b "admin:secret123" \
  --autoreply "Thanks for your message!" \
  -w "https://webhook.site/your-id" \
  --webhook-secret "super-secret"
```

**Using environment variables:**

```bash
docker run -d \
  --name whatsapp \
  --publish 3000:3000 \
  --volume whatsapp-data:/app/storages \
  --restart always \
  --env APP_PORT=3000 \
  --env APP_DEBUG=false \
  --env APP_OS="MyApp" \
  --env APP_BASIC_AUTH="admin:secret123" \
  --env WHATSAPP_AUTO_REPLY="Thanks for your message!" \
  --env WHATSAPP_WEBHOOK="https://webhook.site/your-id" \
  --env WHATSAPP_WEBHOOK_SECRET="super-secret" \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest
```

**Docker Compose:**

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
      # Application Settings
      - APP_PORT=3000
      - APP_DEBUG=false
      - APP_OS=MyApp
      - APP_BASIC_AUTH=admin:secret123,user:pass456

      # WhatsApp Features
      - WHATSAPP_AUTO_REPLY=Thanks for your message!
      - WHATSAPP_AUTO_MARK_READ=true
      - WHATSAPP_WEBHOOK=https://webhook.site/your-id
      - WHATSAPP_WEBHOOK_SECRET=super-secret
      - WHATSAPP_ACCOUNT_VALIDATION=true
      - WHATSAPP_CHAT_STORAGE=true

      # Timezone
      - TZ=America/Sao_Paulo
    command:
      - rest

volumes:
  whatsapp-data:
```

### Systemd Service Configuration

**Using environment file:**

Create `/etc/whatsapp/config.env`:
```bash
APP_PORT=3000
APP_DEBUG=false
APP_OS=MyApp
APP_BASIC_AUTH=admin:secret123
WHATSAPP_WEBHOOK=https://webhook.site/your-id
WHATSAPP_WEBHOOK_SECRET=super-secret
```

Update `/etc/systemd/system/whatsapp.service`:
```ini
[Unit]
Description=WhatsApp Web API
After=network.target

[Service]
Type=simple
User=whatsapp
Group=whatsapp
WorkingDirectory=/opt/whatsapp
EnvironmentFile=/etc/whatsapp/config.env
ExecStart=/opt/whatsapp/whatsapp rest
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Reload and restart:
```bash
sudo systemctl daemon-reload
sudo systemctl restart whatsapp
```

## Configuration Examples

### Development Setup

**Quick local development with debug enabled:**

```bash
./whatsapp rest \
  --port 3000 \
  --debug=true \
  --os="Dev" \
  --chat-storage=true
```

### Production Setup

**Secure production deployment:**

```bash
./whatsapp rest \
  --port 3000 \
  --debug=false \
  --os="Production API" \
  -b "admin:$(openssl rand -base64 32)" \
  --db-uri="postgresql://whatsapp:password@localhost:5432/whatsapp" \
  --webhook="https://api.yourapp.com/webhooks/whatsapp" \
  --webhook-secret="$(openssl rand -hex 32)" \
  --account-validation=true \
  --chat-storage=true
```

### Customer Support Setup

**With auto-reply and webhooks:**

```bash
./whatsapp rest \
  --port 3000 \
  --os="Support Bot" \
  -b "support:secret123" \
  --autoreply="Thanks for contacting support! A team member will respond within 2 hours." \
  --auto-mark-read=true \
  --webhook="https://support.yourapp.com/webhooks/whatsapp" \
  --webhook-secret="your-secret"
```

### Testing Setup

**For testing and development:**

```bash
./whatsapp rest \
  --port 8080 \
  --debug=true \
  --os="Test Instance" \
  --account-validation=false \
  --chat-storage=false
```

### Multi-Instance Setup

**Running multiple instances:**

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

## Configuration Validation

### Check Current Configuration

**View effective configuration:**

```bash
# Check if application is running with correct port
curl -I http://localhost:3000

# Test authentication (if enabled)
curl -u admin:password http://localhost:3000/app/devices

# Check device name
curl http://localhost:3000/user/info

# Test webhook (send test message and check webhook receives it)
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{"phone": "your-number", "message": "Test"}'
```

### Common Configuration Mistakes

**1. Port conflicts:**
```bash
# Check if port is in use
netstat -tulpn | grep 3000
lsof -i :3000

# Use different port
./whatsapp rest --port 8080
```

**2. Authentication format:**
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
DB_URI="postgres://localhost/whatsapp"  # Wrong protocol (use postgresql://)
DB_URI="file:whatsapp.db"               # Missing path

# ✅ Correct
DB_URI="postgresql://user:pass@localhost:5432/whatsapp"
DB_URI="file:storages/whatsapp.db?_foreign_keys=on"
```

## Next Steps

Now that you understand configuration:

1. **[Quick Start Guide](quick-start.md)** - Get started quickly
2. **[First Message Guide](first-message.md)** - Send your first message
3. **[Webhook Documentation](../webhook-payload.md)** - Setup webhook integration
4. **[Deployment Guide](../guides/deployment/)** - Production deployment
5. **[API Documentation](../reference/openapi.yaml)** - Complete API reference

## Quick Reference

### All Configuration Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_PORT` | 3000 | HTTP server port |
| `APP_DEBUG` | false | Debug logging |
| `APP_OS` | Chrome | Device name |
| `APP_BASIC_AUTH` | - | Basic authentication |
| `APP_BASE_PATH` | - | Base URL path |
| `DB_URI` | file:storages/whatsapp.db | Main database |
| `DB_KEYS_URI` | file::memory: | Keys database |
| `WHATSAPP_AUTO_REPLY` | - | Auto-reply message |
| `WHATSAPP_AUTO_MARK_READ` | false | Auto-mark read |
| `WHATSAPP_WEBHOOK` | - | Webhook URL(s) |
| `WHATSAPP_WEBHOOK_SECRET` | secret | HMAC secret |
| `WHATSAPP_ACCOUNT_VALIDATION` | true | Validate accounts |
| `WHATSAPP_CHAT_STORAGE` | true | Store chat history |

### Command-Line Flags

```bash
# View all available flags
./whatsapp rest --help

# Common flags
--port          HTTP port (default: 3000)
--debug         Debug mode (default: false)
--os            Device name (default: Chrome)
-b, --basic-auth    Basic auth credentials
--base-path     Base URL path
--autoreply     Auto-reply message
--auto-mark-read    Auto-mark messages as read
-w, --webhook   Webhook URL(s)
--webhook-secret    Webhook HMAC secret
--db-uri        Database connection string
--account-validation    Validate phone numbers
--chat-storage  Enable chat history storage
```

---

**Version**: Compatible with v7.7.0+
**Last Updated**: 2025-11-14
