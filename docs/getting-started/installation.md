# Installation Guide

Comprehensive installation instructions for all platforms and deployment methods.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installing FFmpeg](#installing-ffmpeg)
- [Installation Methods](#installation-methods)
  - [Pre-built Binary](#method-1-pre-built-binary-recommended)
  - [Docker](#method-2-docker)
  - [Build from Source](#method-3-build-from-source)
- [Platform-Specific Instructions](#platform-specific-instructions)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required

- **Operating System**: Linux, macOS, or Windows
- **Architecture**: AMD64 or ARM64
- **WhatsApp Account**: Active WhatsApp account with phone
- **FFmpeg**: Required for media processing

### Recommended

- **Reverse Proxy**: For production (nginx, Caddy, Traefik)
- **PostgreSQL**: For production database (SQLite is default)

### System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 1 core | 2+ cores |
| RAM | 512 MB | 1 GB+ |
| Disk | 100 MB | 500 MB+ |
| Network | 1 Mbps | 10+ Mbps |

## Installing FFmpeg

FFmpeg is required for processing media files (images, videos, audio).

### macOS

**Using Homebrew (recommended):**
```bash
brew install ffmpeg
```

**Verify installation:**
```bash
ffmpeg -version
# Expected: ffmpeg version 6.0 or higher
```

**Special note for macOS:**
If you encounter `invalid flag in pkg-config --cflags: -Xpreprocessor` error:
```bash
export CGO_CFLAGS_ALLOW="-Xpreprocessor"
```

### Ubuntu/Debian

```bash
# Update package list
sudo apt update

# Install FFmpeg
sudo apt install ffmpeg -y

# Verify installation
ffmpeg -version
```

### CentOS/RHEL/Fedora

```bash
# Enable EPEL repository (CentOS/RHEL)
sudo yum install epel-release -y

# Install FFmpeg
sudo yum install ffmpeg -y

# Or using dnf (Fedora)
sudo dnf install ffmpeg -y

# Verify installation
ffmpeg -version
```

### Alpine Linux (Docker)

```bash
# Install FFmpeg
apk add --no-cache ffmpeg

# Verify installation
ffmpeg -version
```

### Windows

**Option 1: Chocolatey (recommended)**
```powershell
# Install Chocolatey if not already installed
# https://chocolatey.org/install

# Install FFmpeg
choco install ffmpeg -y

# Verify installation
ffmpeg -version
```

**Option 2: Manual Installation**

1. Download FFmpeg from [ffmpeg.org](https://ffmpeg.org/download.html#build-windows)
2. Extract the archive
3. Add the `bin` directory to system PATH:
   - Right-click **This PC** > **Properties**
   - Click **Advanced system settings**
   - Click **Environment Variables**
   - Under **System variables**, find and select **Path**
   - Click **Edit** > **New**
   - Add the path to FFmpeg `bin` directory (e.g., `C:\ffmpeg\bin`)
   - Click **OK** on all dialogs
4. Verify installation:
   ```powershell
   ffmpeg -version
   ```

**Note:** For Windows, using WSL (Windows Subsystem for Linux) is recommended for better compatibility.

## Installation Methods

### Method 1: Pre-built Binary (Recommended)

Download pre-compiled binaries for your platform from GitHub releases.

#### Linux AMD64

```bash
# Create application directory
mkdir -p /opt/whatsapp
cd /opt/whatsapp

# Download latest binary
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-linux-amd64

# Make executable
chmod +x whatsapp-linux-amd64

# Rename for convenience
mv whatsapp-linux-amd64 whatsapp

# Create directories
mkdir -p storages statics/media logs

# Run
./whatsapp rest
```

#### Linux ARM64

```bash
# Create application directory
mkdir -p /opt/whatsapp
cd /opt/whatsapp

# Download latest binary
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-linux-arm64

# Make executable
chmod +x whatsapp-linux-arm64

# Rename for convenience
mv whatsapp-linux-arm64 whatsapp

# Create directories
mkdir -p storages statics/media logs

# Run
./whatsapp rest
```

#### macOS Intel (AMD64)

```bash
# Create application directory
mkdir -p ~/whatsapp
cd ~/whatsapp

# Download latest binary
curl -L -o whatsapp https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-darwin-amd64

# Make executable
chmod +x whatsapp

# Create directories
mkdir -p storages statics/media logs

# Run
./whatsapp rest
```

#### macOS Apple Silicon (ARM64)

```bash
# Create application directory
mkdir -p ~/whatsapp
cd ~/whatsapp

# Download latest binary
curl -L -o whatsapp https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-darwin-arm64

# Make executable
chmod +x whatsapp

# Create directories
mkdir -p storages statics/media logs

# Run
./whatsapp rest
```

#### Windows

**PowerShell:**
```powershell
# Create application directory
New-Item -ItemType Directory -Path C:\whatsapp
Set-Location C:\whatsapp

# Download latest binary
Invoke-WebRequest -Uri https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-windows-amd64.exe -OutFile whatsapp.exe

# Create directories
New-Item -ItemType Directory -Path storages
New-Item -ItemType Directory -Path statics\media
New-Item -ItemType Directory -Path logs

# Run
.\whatsapp.exe rest
```

**Note:** You may need to allow the executable through Windows Firewall and SmartScreen.

### Method 2: Docker

Docker provides the easiest deployment method with all dependencies included.

#### Quick Start with Docker

```bash
# Pull and run latest image
docker run -d \
  --name whatsapp \
  --publish 3000:3000 \
  --volume whatsapp-data:/app/storages \
  --restart always \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest
```

#### Docker with Configuration

```bash
# Run with custom configuration
docker run -d \
  --name whatsapp \
  --publish 3000:3000 \
  --volume whatsapp-data:/app/storages \
  --restart always \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest \
  --basic-auth="admin:secret123" \
  --debug=true \
  --os="MyApp"
```

#### Docker Compose

**Create `docker-compose.yml`:**
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
    command:
      - rest
      - --port=3000
      - --debug=false
      - --os=MyApp
      - --basic-auth=admin:secret123
    environment:
      - TZ=America/Sao_Paulo

volumes:
  whatsapp-data:
```

**Start services:**
```bash
# Start in detached mode
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Stop and remove volumes (deletes all data)
docker-compose down -v
```

#### Docker Compose with Environment Variables

**Create `docker-compose.yml`:**
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
      - APP_PORT=3000
      - APP_DEBUG=false
      - APP_OS=MyApp
      - APP_BASIC_AUTH=admin:secret123
      - WHATSAPP_AUTO_REPLY=Thanks for your message!
      - WHATSAPP_WEBHOOK=https://webhook.site/your-id
      - WHATSAPP_WEBHOOK_SECRET=super-secret
      - TZ=America/Sao_Paulo
    command:
      - rest

volumes:
  whatsapp-data:
```

**Create `.env` file (optional):**
```bash
APP_PORT=3000
APP_DEBUG=false
APP_OS=MyApp
APP_BASIC_AUTH=admin:secret123
WHATSAPP_WEBHOOK=https://webhook.site/your-id
```

**Start services:**
```bash
docker-compose up -d
```

### Method 3: Build from Source

Build the application from source code for development or customization.

#### Prerequisites for Building

- **Go**: Version 1.24.0 or higher
- **Git**: For cloning repository
- **Make**: Optional, for using Makefile
- **FFmpeg**: For media processing

#### Install Go

**macOS:**
```bash
brew install go
```

**Ubuntu/Debian:**
```bash
# Download and install Go
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

**Windows:**
Download installer from [go.dev](https://go.dev/dl/) and follow installation wizard.

#### Build Steps

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

# Or build with optimizations
go build -ldflags="-s -w" -o whatsapp

# Run
./whatsapp rest
```

#### Cross-Compilation

Build for different platforms from your development machine:

```bash
cd src

# Build for Linux AMD64
GOOS=linux GOARCH=amd64 go build -o whatsapp-linux-amd64

# Build for Linux ARM64
GOOS=linux GOARCH=arm64 go build -o whatsapp-linux-arm64

# Build for macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o whatsapp-darwin-amd64

# Build for macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o whatsapp-darwin-arm64

# Build for Windows AMD64
GOOS=windows GOARCH=amd64 go build -o whatsapp-windows-amd64.exe
```

#### Build with Version Information

```bash
VERSION=v7.8.3
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD)

go build -ldflags="-s -w \
  -X main.Version=$VERSION \
  -X main.BuildDate=$BUILD_DATE \
  -X main.GitCommit=$GIT_COMMIT" \
  -o whatsapp
```

## Platform-Specific Instructions

### Linux Server Deployment

#### Create System User

```bash
# Create dedicated user for running application
sudo useradd -r -s /bin/false -d /opt/whatsapp whatsapp

# Create application directory
sudo mkdir -p /opt/whatsapp
sudo chown -R whatsapp:whatsapp /opt/whatsapp
```

#### Install Binary

```bash
# Download and install
sudo -u whatsapp bash << 'EOF'
cd /opt/whatsapp
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-linux-amd64
chmod +x whatsapp-linux-amd64
mv whatsapp-linux-amd64 whatsapp
mkdir -p storages statics/media logs
EOF
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
```

### macOS Local Development

#### Using Homebrew Tap (Future)

```bash
# This will be available in future releases
brew tap chatwoot-br/whatsapp
brew install whatsapp
```

#### Manual Installation

```bash
# Create application directory
mkdir -p ~/Applications/whatsapp
cd ~/Applications/whatsapp

# Download binary
curl -L -o whatsapp https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-darwin-arm64

# Make executable
chmod +x whatsapp

# Remove quarantine attribute (if needed)
xattr -d com.apple.quarantine whatsapp 2>/dev/null || true

# Create directories
mkdir -p storages statics/media logs

# Run
./whatsapp rest
```

#### Create Launch Agent (Auto-start on Login)

Create `~/Library/LaunchAgents/com.whatsapp.api.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.whatsapp.api</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/YOUR_USERNAME/Applications/whatsapp/whatsapp</string>
        <string>rest</string>
        <string>--port</string>
        <string>3000</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/Users/YOUR_USERNAME/Applications/whatsapp</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/YOUR_USERNAME/Applications/whatsapp/logs/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/YOUR_USERNAME/Applications/whatsapp/logs/stderr.log</string>
</dict>
</plist>
```

**Load and start:**
```bash
# Load agent
launchctl load ~/Library/LaunchAgents/com.whatsapp.api.plist

# Start service
launchctl start com.whatsapp.api

# Check status
launchctl list | grep whatsapp

# Stop service
launchctl stop com.whatsapp.api

# Unload agent
launchctl unload ~/Library/LaunchAgents/com.whatsapp.api.plist
```

### Windows Installation

#### Using WSL (Recommended)

Windows Subsystem for Linux provides better compatibility:

```powershell
# Install WSL2
wsl --install

# Open WSL terminal and follow Linux installation instructions
wsl
```

#### Native Windows Installation

```powershell
# Create application directory
New-Item -ItemType Directory -Path C:\whatsapp -Force
Set-Location C:\whatsapp

# Download binary
$url = "https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-windows-amd64.exe"
Invoke-WebRequest -Uri $url -OutFile whatsapp.exe

# Create directories
New-Item -ItemType Directory -Path storages -Force
New-Item -ItemType Directory -Path statics\media -Force
New-Item -ItemType Directory -Path logs -Force

# Run
.\whatsapp.exe rest
```

#### Create Windows Service

Using NSSM (Non-Sucking Service Manager):

```powershell
# Download and install NSSM
choco install nssm -y

# Install service
nssm install WhatsAppAPI "C:\whatsapp\whatsapp.exe" rest --port 3000

# Set service properties
nssm set WhatsAppAPI AppDirectory C:\whatsapp
nssm set WhatsAppAPI DisplayName "WhatsApp Web API"
nssm set WhatsAppAPI Description "WhatsApp Web API Service"
nssm set WhatsAppAPI Start SERVICE_AUTO_START

# Start service
nssm start WhatsAppAPI

# Check status
nssm status WhatsAppAPI
```

## Verification

### Verify Installation

```bash
# Check binary version (if supported)
./whatsapp --version

# Check binary information
file whatsapp

# Check dependencies
ldd whatsapp  # Linux only
```

### Test Run

```bash
# Run with debug to verify configuration
./whatsapp rest --debug=true --port 8080

# In another terminal, test API
curl http://localhost:8080/app/devices
```

### Check Access

```bash
# Test HTTP endpoint
curl -I http://localhost:3000

# Expected response
HTTP/1.1 200 OK
Content-Type: text/html; charset=utf-8
...
```

## Troubleshooting

### Binary Not Executing

**Problem:** Permission denied or command not found

**Solution:**
```bash
# Make executable
chmod +x whatsapp

# Check file type
file whatsapp

# Check architecture matches
uname -m
# Should match binary architecture (x86_64 or aarch64)
```

### Port Already in Use

**Problem:** Address already in use

**Solution:**
```bash
# Check what's using the port
netstat -tulpn | grep 3000
lsof -i :3000

# Kill process or use different port
./whatsapp rest --port 8080
```

### FFmpeg Not Found

**Problem:** Media processing fails

**Solution:**
```bash
# Verify FFmpeg installation
which ffmpeg
ffmpeg -version

# Install if missing (see FFmpeg section above)

# Check PATH
echo $PATH

# Add FFmpeg to PATH if needed
export PATH=$PATH:/path/to/ffmpeg
```

### Permission Issues

**Problem:** Cannot write to directories

**Solution:**
```bash
# Check directory permissions
ls -la storages/

# Fix permissions
chmod 755 storages statics
chmod 644 storages/*.db

# Check ownership
sudo chown -R $(whoami):$(whoami) .
```

### Database Errors

**Problem:** Cannot open database

**Solution:**
```bash
# Check if database is locked
lsof storages/whatsapp.db

# Remove lock file if exists
rm -f storages/whatsapp.db-*

# Verify database permissions
chmod 644 storages/whatsapp.db
```

### macOS Quarantine Issues

**Problem:** "whatsapp cannot be opened because the developer cannot be verified"

**Solution:**
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine whatsapp

# Or allow in System Preferences
# Go to System Preferences > Security & Privacy > General
# Click "Open Anyway"
```

### SELinux Issues (Linux)

**Problem:** Permission denied on SELinux-enabled systems

**Solution:**
```bash
# Check SELinux status
getenforce

# If enforcing, configure contexts
sudo semanage fcontext -a -t bin_t "/opt/whatsapp/whatsapp"
sudo semanage fcontext -a -t httpd_sys_rw_content_t "/opt/whatsapp/storages(/.*)?"
sudo restorecon -R /opt/whatsapp
```

## Next Steps

After successful installation:

1. **[Quick Start Guide](quick-start.md)** - Get up and running quickly
2. **[Configuration Basics](configuration-basics.md)** - Configure the application
3. **[First Message Guide](first-message.md)** - Send your first message
4. **[Deployment Guide](../guides/deployment/)** - Production deployment options

## Additional Resources

- **[GitHub Releases](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases)** - Download binaries
- **[Docker Hub](https://ghcr.io/chatwoot-br/go-whatsapp-web-multidevice)** - Docker images
- **[API Documentation](../reference/openapi.yaml)** - REST API reference

---

**Version**: Compatible with v7.7.0+
**Last Updated**: 2025-11-14
