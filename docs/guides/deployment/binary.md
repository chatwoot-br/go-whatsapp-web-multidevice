# Binary Deployment Guide

This guide covers deploying the WhatsApp Web API Multidevice application using pre-built binaries or building from source.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation Methods](#installation-methods)
- [Initial Setup](#initial-setup)
- [Configuration](#configuration)
- [Running the Application](#running-the-application)
- [Deployment Scenarios](#deployment-scenarios)
- [Troubleshooting](#troubleshooting)
- [Related Guides](#related-guides)

## Prerequisites

### Required

- **Operating System**: Linux, macOS, or Windows
- **Architecture**: AMD64 or ARM64
- **WhatsApp Account**: Active WhatsApp account with phone

### Optional

- **FFmpeg**: Required for media compression (video/image processing)
- **PostgreSQL**: For production database (SQLite is default)
- **Reverse Proxy**: For HTTPS and production deployment (nginx, Caddy, Traefik)
- **Go**: Version 1.21 or higher (only for building from source)

### Installing FFmpeg

FFmpeg is required for processing media files (images, videos, audio).

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install ffmpeg
```

**macOS:**
```bash
brew install ffmpeg
```

**Windows:**
Download from [ffmpeg.org](https://ffmpeg.org/download.html) and add to PATH

**Verify installation:**
```bash
ffmpeg -version
```

## Installation Methods

### Method 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform from GitHub:

**Linux AMD64:**
```bash
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-linux-amd64
chmod +x whatsapp-linux-amd64
mv whatsapp-linux-amd64 whatsapp
```

**Linux ARM64:**
```bash
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-linux-arm64
chmod +x whatsapp-linux-arm64
mv whatsapp-linux-arm64 whatsapp
```

**macOS AMD64 (Intel):**
```bash
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-darwin-amd64
chmod +x whatsapp-darwin-amd64
mv whatsapp-darwin-amd64 whatsapp
```

**macOS ARM64 (Apple Silicon):**
```bash
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-darwin-arm64
chmod +x whatsapp-darwin-arm64
mv whatsapp-darwin-arm64 whatsapp
```

**Windows:**
```powershell
# Download from releases page
# https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest
# whatsapp-windows-amd64.exe
```

### Method 2: Build from Source

**Prerequisites for building:**
- Go 1.21 or higher
- Git

**Build steps:**

```bash
# Clone repository
git clone https://github.com/chatwoot-br/go-whatsapp-web-multidevice.git
cd go-whatsapp-web-multidevice

# Navigate to source directory
cd src

# Download dependencies
go mod download

# Build binary
go build -o whatsapp

# Or for specific platform
GOOS=linux GOARCH=amd64 go build -o whatsapp-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o whatsapp-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o whatsapp.exe

# Run
./whatsapp rest
```

**Build with optimizations:**

```bash
# Build with version info and optimizations
go build -ldflags="-s -w -X main.Version=v7.8.3" -o whatsapp
```

## Initial Setup

### 1. Create Directory Structure

Create required directories for the application:

```bash
# Create application directory
mkdir -p /opt/whatsapp
cd /opt/whatsapp

# Create subdirectories
mkdir -p storages statics/media statics/qrcode logs
```

### 2. Place Binary

```bash
# Copy binary to application directory
cp whatsapp /opt/whatsapp/
chmod +x /opt/whatsapp/whatsapp
```

### 3. Create Configuration File (Optional)

Create `src/.env` file for configuration (can also use CLI flags):

```bash
# Application Settings
APP_PORT=3000
APP_DEBUG=false
APP_OS=MyAppName
APP_BASIC_AUTH=user1:pass1,user2:pass2
APP_BASE_PATH=

# Database Settings
DB_URI="file:storages/whatsapp.db?_foreign_keys=on"
DB_KEYS_URI="file::memory:?cache=shared&_foreign_keys=on"

# WhatsApp Settings
WHATSAPP_AUTO_REPLY="Thanks for your message!"
WHATSAPP_AUTO_MARK_READ=false
WHATSAPP_WEBHOOK=https://your-webhook.com/handler
WHATSAPP_WEBHOOK_SECRET=super-secret-key
WHATSAPP_ACCOUNT_VALIDATION=true
WHATSAPP_CHAT_STORAGE=true
```

### 4. Set Permissions (Linux/macOS)

```bash
# Set executable permissions
chmod +x /opt/whatsapp/whatsapp

# Set directory permissions
chmod 755 /opt/whatsapp/storages
chmod 755 /opt/whatsapp/statics

# If running as specific user
sudo chown -R whatsapp:whatsapp /opt/whatsapp
```

## Configuration

### Configuration Priority

Configuration is loaded in this order (later overrides earlier):

1. **`.env` file** (lowest priority)
2. **Environment variables**
3. **Command-line flags** (highest priority)

### Key Configuration Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--port` | `APP_PORT` | 3000 | HTTP server port |
| `--debug` | `APP_DEBUG` | false | Enable debug logging |
| `--os` | `APP_OS` | Chrome | Device name in WhatsApp |
| `-b, --basic-auth` | `APP_BASIC_AUTH` | - | Basic auth (user:pass,user2:pass2) |
| `--base-path` | `APP_BASE_PATH` | - | Base path for subpath deployment |
| `--autoreply` | `WHATSAPP_AUTO_REPLY` | - | Auto-reply message |
| `--auto-mark-read` | `WHATSAPP_AUTO_MARK_READ` | false | Auto-mark messages as read |
| `-w, --webhook` | `WHATSAPP_WEBHOOK` | - | Webhook URLs (comma-separated) |
| `--webhook-secret` | `WHATSAPP_WEBHOOK_SECRET` | secret | HMAC secret for webhooks |
| `--db-uri` | `DB_URI` | file:storages/whatsapp.db | Main database URI |

### Configuration Examples

**Using environment variables:**
```bash
export APP_PORT=8080
export APP_BASIC_AUTH="admin:secret123"
export WHATSAPP_WEBHOOK="https://webhook.site/your-id"
./whatsapp rest
```

**Using command-line flags:**
```bash
./whatsapp rest --port 8080 -b admin:secret123 -w https://webhook.site/your-id
```

**Using .env file:**
```bash
# Create .env file in working directory
cat > .env <<EOF
APP_PORT=8080
APP_BASIC_AUTH=admin:secret123
WHATSAPP_WEBHOOK=https://webhook.site/your-id
EOF

# Run application (automatically reads .env)
./whatsapp rest
```

## Running the Application

### REST API Mode

**Basic usage:**
```bash
# Run with defaults
./whatsapp rest

# Access at http://localhost:3000
```

**With configuration:**
```bash
# Basic with custom port
./whatsapp rest --port 8080

# With authentication
./whatsapp rest -b admin:secret123

# With webhook
./whatsapp rest -w https://webhook.site/your-id --webhook-secret mysecret

# With debug logging
./whatsapp rest --debug true
```

**Full example:**
```bash
./whatsapp rest \
  --port 3000 \
  --debug false \
  --os "MyApp" \
  -b "admin:secret123" \
  --autoreply "Thanks for your message!" \
  -w "https://webhook.site/your-id" \
  --webhook-secret "super-secret"
```

### MCP Mode (Model Context Protocol)

**Basic usage:**
```bash
# Run MCP server
./whatsapp mcp

# Default port: 8080
```

**With configuration:**
```bash
# Custom port and host
./whatsapp mcp --port 8080 --host 0.0.0.0

# With debug logging
./whatsapp mcp --debug true
```

**MCP Endpoints:**
- SSE: `http://localhost:8080/sse`
- Message: `http://localhost:8080/message`

### Login to WhatsApp

#### Option 1: QR Code (Web UI)

1. Start the application: `./whatsapp rest`
2. Open browser: `http://localhost:3000`
3. Navigate to Login page
4. Scan QR code with WhatsApp mobile app
5. Wait for connection confirmation

#### Option 2: QR Code (API)

```bash
# Get QR code
curl -X GET http://localhost:3000/app/login \
  -u admin:secret123

# Response includes QR image URL
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "qr_link": "http://localhost:3000/statics/images/qrcode/scan-qr-xxx.png",
    "qr_duration": 30
  }
}
```

#### Option 3: Pairing Code (API)

```bash
# Get pairing code (include country code)
curl -X GET "http://localhost:3000/app/login-with-code?phone=628912344551" \
  -u admin:secret123

# Response
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "pair_code": "ABCD-1234"
  }
}

# Enter this code in WhatsApp mobile app:
# Settings > Linked Devices > Link a Device > Link with phone number instead
```

## Deployment Scenarios

### Scenario 1: Local Development

Quick start for development and testing:

```bash
# Start with debug logging
./whatsapp rest --debug true

# Access at http://localhost:3000
# Login via QR code in web interface
```

### Scenario 2: Server Deployment with Systemd

Deploy as a system service for automatic startup and management.

#### Create System User

```bash
# Create dedicated user
sudo useradd -r -s /bin/false whatsapp

# Create application directory
sudo mkdir -p /opt/whatsapp
sudo cp whatsapp /opt/whatsapp/
sudo mkdir -p /opt/whatsapp/storages /opt/whatsapp/statics /opt/whatsapp/logs

# Set ownership
sudo chown -R whatsapp:whatsapp /opt/whatsapp
```

#### Create Systemd Service

Create `/etc/systemd/system/whatsapp.service`:

```ini
[Unit]
Description=WhatsApp Web API
After=network.target

[Service]
Type=simple
User=whatsapp
Group=whatsapp
WorkingDirectory=/opt/whatsapp
ExecStart=/opt/whatsapp/whatsapp rest --port 3000
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=whatsapp

# Environment variables (optional)
Environment="APP_BASIC_AUTH=admin:secret123"
Environment="WHATSAPP_WEBHOOK=https://your-webhook.com/handler"
Environment="WHATSAPP_WEBHOOK_SECRET=your-secret"

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/whatsapp/storages /opt/whatsapp/logs

[Install]
WantedBy=multi-user.target
```

#### Enable and Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service (start on boot)
sudo systemctl enable whatsapp

# Start service
sudo systemctl start whatsapp

# Check status
sudo systemctl status whatsapp

# View logs
sudo journalctl -u whatsapp -f

# Stop service
sudo systemctl stop whatsapp

# Restart service
sudo systemctl restart whatsapp
```

#### Service Management

```bash
# View real-time logs
sudo journalctl -u whatsapp -f

# View recent logs
sudo journalctl -u whatsapp -n 100

# View logs since boot
sudo journalctl -u whatsapp -b

# Check service status
sudo systemctl status whatsapp

# Reload service configuration
sudo systemctl daemon-reload
sudo systemctl restart whatsapp
```

### Scenario 3: Subpath Deployment

Deploy under a specific path (e.g., `/whatsapp-api`) for integration with existing web services:

```bash
# Run with base path
./whatsapp rest --base-path="/whatsapp-api" --port 3000

# API accessible at:
# - Base: http://localhost:3000/whatsapp-api/
# - Endpoints: http://localhost:3000/whatsapp-api/send/message
# - Web UI: http://localhost:3000/whatsapp-api/
```

**Nginx reverse proxy configuration:**

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    location /whatsapp-api/ {
        proxy_pass http://localhost:3000/whatsapp-api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket support
    location /whatsapp-api/ws {
        proxy_pass http://localhost:3000/whatsapp-api/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Scenario 4: Multiple Instances (Multiple Accounts)

Run separate instances for different WhatsApp accounts:

```bash
# Account 1 on port 3001
./whatsapp rest \
  --port 3001 \
  --db-uri "file:storages/account1.db" \
  -b "admin1:pass1"

# Account 2 on port 3002
./whatsapp rest \
  --port 3002 \
  --db-uri "file:storages/account2.db" \
  -b "admin2:pass2"

# Account 3 on port 3003
./whatsapp rest \
  --port 3003 \
  --db-uri "file:storages/account3.db" \
  -b "admin3:pass3"
```

**Systemd services for multiple accounts:**

Create `/etc/systemd/system/whatsapp@.service`:

```ini
[Unit]
Description=WhatsApp Web API - Instance %i
After=network.target

[Service]
Type=simple
User=whatsapp
WorkingDirectory=/opt/whatsapp/%i
ExecStart=/opt/whatsapp/whatsapp rest --port 300%i --db-uri "file:storages/whatsapp.db"
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Start multiple instances:

```bash
# Enable and start instance 1
sudo systemctl enable whatsapp@1
sudo systemctl start whatsapp@1

# Enable and start instance 2
sudo systemctl enable whatsapp@2
sudo systemctl start whatsapp@2

# Check status
sudo systemctl status whatsapp@1
sudo systemctl status whatsapp@2
```

### Scenario 5: Production with PostgreSQL

Use PostgreSQL for better performance and reliability:

```bash
# Setup PostgreSQL database
sudo -u postgres psql
CREATE DATABASE whatsapp;
CREATE USER whatsapp WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE whatsapp TO whatsapp;
\q

# Run with PostgreSQL
./whatsapp rest \
  --port 3000 \
  -b "admin:secret123" \
  --db-uri "postgresql://whatsapp:your-password@localhost:5432/whatsapp?sslmode=disable"
```

## Troubleshooting

### Application Won't Start

**Problem**: Binary fails to execute

**Solution**:

```bash
# Check binary exists and is executable
ls -la whatsapp
chmod +x whatsapp

# Check architecture matches
file whatsapp
uname -m

# Run with debug to see errors
./whatsapp rest --debug true

# Check if port is already in use
netstat -tulpn | grep 3000
lsof -i :3000

# Try different port
./whatsapp rest --port 8080
```

### Permission Denied Errors

**Problem**: Cannot write to directories

**Solution**:

```bash
# Check directory permissions
ls -la storages/ statics/

# Fix permissions
chmod 755 storages statics
chmod 644 storages/*.db

# If running as specific user
sudo chown -R whatsapp:whatsapp /opt/whatsapp

# Check SELinux (if enabled)
getenforce
sudo semanage fcontext -a -t httpd_sys_rw_content_t "/opt/whatsapp/storages(/.*)?"
sudo restorecon -R /opt/whatsapp/storages
```

### Database Issues

**Problem**: Database locked or corrupted

**Solution**:

```bash
# Stop application
sudo systemctl stop whatsapp  # or kill process

# Backup database
cp storages/whatsapp.db storages/whatsapp.db.backup

# Check database integrity
sqlite3 storages/whatsapp.db "PRAGMA integrity_check;"

# If corrupted, restore from backup or start fresh
rm storages/whatsapp.db
# Restart application and login again
```

### FFmpeg Not Found

**Problem**: Media processing fails

**Solution**:

```bash
# Check FFmpeg installation
ffmpeg -version

# Install FFmpeg
# Ubuntu/Debian
sudo apt install ffmpeg

# macOS
brew install ffmpeg

# Verify after installation
which ffmpeg
ffmpeg -version
```

### Connection Issues

**Problem**: Cannot connect to WhatsApp

**Solution**:

```bash
# Check device status
curl http://localhost:3000/app/devices -u admin:secret123

# Try reconnect
curl http://localhost:3000/app/reconnect -u admin:secret123

# Logout and re-login
curl http://localhost:3000/app/logout -u admin:secret123
# Then login again via QR or pairing code

# Check logs
tail -f logs/app.log  # if logging to file
sudo journalctl -u whatsapp -f  # if using systemd
```

### High Memory Usage

**Problem**: Application consuming too much memory

**Solution**:

```bash
# Disable chat storage if not needed
./whatsapp rest --chat-storage=false

# Monitor memory usage
ps aux | grep whatsapp
top -p $(pgrep whatsapp)

# Check database size
du -sh storages/

# Consider using PostgreSQL for better memory management
./whatsapp rest --db-uri "postgresql://user:pass@localhost/whatsapp"
```

## Process Management

### Running in Background

**Using screen:**
```bash
# Start new screen session
screen -S whatsapp

# Run application
./whatsapp rest

# Detach: Ctrl+A, then D
# Reattach: screen -r whatsapp
```

**Using tmux:**
```bash
# Start new tmux session
tmux new -s whatsapp

# Run application
./whatsapp rest

# Detach: Ctrl+B, then D
# Reattach: tmux attach -t whatsapp
```

**Using nohup:**
```bash
# Run in background
nohup ./whatsapp rest > logs/whatsapp.log 2>&1 &

# Check process
ps aux | grep whatsapp

# Kill process
pkill whatsapp
```

### Monitoring

**Check process status:**
```bash
# Check if running
ps aux | grep whatsapp
pgrep whatsapp

# Check resource usage
top -p $(pgrep whatsapp)
htop -p $(pgrep whatsapp)

# Check open files
lsof -p $(pgrep whatsapp)

# Check network connections
netstat -tulpn | grep $(pgrep whatsapp)
```

## Backup and Restore

### Manual Backup

```bash
# Stop application
sudo systemctl stop whatsapp

# Backup databases
cp storages/whatsapp.db storages/whatsapp.db.backup
cp storages/chatstorage.db storages/chatstorage.db.backup

# Or backup entire directory
tar czf whatsapp-backup-$(date +%Y%m%d).tar.gz storages/

# Start application
sudo systemctl start whatsapp
```

### Automated Backup Script

Create `backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/backup/whatsapp"
APP_DIR="/opt/whatsapp"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup databases
cp "$APP_DIR/storages/whatsapp.db" "$BACKUP_DIR/whatsapp-$DATE.db"
cp "$APP_DIR/storages/chatstorage.db" "$BACKUP_DIR/chatstorage-$DATE.db"

# Or backup entire storages directory
tar czf "$BACKUP_DIR/whatsapp-full-$DATE.tar.gz" -C "$APP_DIR" storages/

# Keep only last 7 backups
cd "$BACKUP_DIR"
ls -t whatsapp-*.db | tail -n +8 | xargs -r rm
ls -t whatsapp-full-*.tar.gz | tail -n +8 | xargs -r rm

echo "Backup completed: $DATE"
```

Make executable and schedule:

```bash
# Make executable
chmod +x backup.sh

# Add to crontab (daily at 2 AM)
crontab -e
# Add line:
0 2 * * * /opt/whatsapp/backup.sh
```

### Restore from Backup

```bash
# Stop application
sudo systemctl stop whatsapp

# Restore database
cp storages/whatsapp.db.backup storages/whatsapp.db

# Or restore from tar
tar xzf whatsapp-backup-20250105.tar.gz -C /opt/whatsapp/

# Set permissions
sudo chown -R whatsapp:whatsapp /opt/whatsapp/storages

# Start application
sudo systemctl start whatsapp
```

## Related Guides

- **[Docker Deployment Guide](docker.md)** - Deploy using Docker and Docker Compose
- **[Kubernetes Deployment Guide](kubernetes.md)** - Deploy on Kubernetes
- **[Production Checklist](production-checklist.md)** - Production deployment best practices
- **[Main Deployment Guide](../../deployment-guide.md)** - Overview of all deployment methods

## Additional Resources

- **API Documentation**: `docs/openapi.yaml` - Full REST API specification
- **Webhook Guide**: `docs/webhook-payload.md` - Webhook integration
- **GitHub Repository**: [go-whatsapp-web-multidevice](https://github.com/chatwoot-br/go-whatsapp-web-multidevice)

---

**Version**: Compatible with v7.7.0+
**Last Updated**: 2025-10-05
