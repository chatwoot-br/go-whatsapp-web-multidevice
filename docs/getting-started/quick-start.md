# Quick Start Guide

Get up and running with the WhatsApp Web API in 5 minutes.

## Prerequisites

Before you begin, ensure you have:

- **Active WhatsApp Account** with a phone
- **FFmpeg** installed (for media processing)
- **Operating System**: Linux, macOS, or Windows

## Step 1: Install FFmpeg

FFmpeg is required for processing images, videos, and audio files.

**macOS:**
```bash
brew install ffmpeg
```

**Ubuntu/Debian:**
```bash
sudo apt update && sudo apt install ffmpeg
```

**Windows:**
- Download from [ffmpeg.org](https://ffmpeg.org/download.html)
- Add to system PATH

**Verify installation:**
```bash
ffmpeg -version
```

## Step 2: Choose Your Installation Method

### Option A: Docker (Recommended for Quick Start)

**1. Run with Docker:**
```bash
docker run -d \
  --name whatsapp \
  --publish 3000:3000 \
  --volume whatsapp-data:/app/storages \
  --restart always \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest
```

**2. Access the application:**
```bash
# Open in browser
open http://localhost:3000
```

**3. View logs (optional):**
```bash
docker logs -f whatsapp
```

### Option B: Pre-built Binary

**1. Download for your platform:**

**Linux (AMD64):**
```bash
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-linux-amd64
chmod +x whatsapp-linux-amd64
mv whatsapp-linux-amd64 whatsapp
```

**macOS (Apple Silicon):**
```bash
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-darwin-arm64
chmod +x whatsapp-darwin-arm64
mv whatsapp-darwin-arm64 whatsapp
```

**macOS (Intel):**
```bash
wget https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases/latest/download/whatsapp-darwin-amd64
chmod +x whatsapp-darwin-amd64
mv whatsapp-darwin-amd64 whatsapp
```

**2. Run the application:**
```bash
./whatsapp rest
```

**3. Access the application:**
```bash
# Open in browser
open http://localhost:3000
```

## Step 3: Login to WhatsApp

### Method 1: QR Code (Web Interface)

1. Open `http://localhost:3000` in your browser
2. Navigate to the **Login** page
3. A QR code will be displayed
4. Open WhatsApp on your phone
5. Go to **Settings** > **Linked Devices** > **Link a Device**
6. Scan the QR code
7. Wait for connection confirmation

### Method 2: Pairing Code (API)

**1. Get pairing code:**
```bash
curl "http://localhost:3000/app/login-with-code?phone=5511999998888"
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "pair_code": "ABCD-1234"
  }
}
```

**2. Enter code in WhatsApp:**
- Open WhatsApp on your phone
- Go to **Settings** > **Linked Devices** > **Link a Device**
- Tap **Link with phone number instead**
- Enter the pairing code: `ABCD-1234`

**Note:** Include country code in phone number (e.g., 5511999998888 for Brazil)

## Step 4: Send Your First Message

### Using the Web Interface

1. Navigate to **Send Message** in the web interface
2. Enter recipient's phone number with country code (e.g., 5511999998888)
3. Type your message
4. Click **Send**

### Using curl (API)

```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Hello from WhatsApp API!"
  }'
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "message_id": "3EB0C431D4D2E2D2F3E8",
    "status": "sent"
  }
}
```

## Step 5: Verify Connection

**Check connection status:**

**Web Interface:**
- Navigate to **Devices** page
- You should see your connected device

**API:**
```bash
curl http://localhost:3000/app/devices
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Success",
  "results": [
    {
      "device": "Chrome (MyApp)",
      "platform": "Chrome",
      "connected": true
    }
  ]
}
```

## Common Quick Start Issues

### Port Already in Use

**Problem:** Port 3000 is already in use

**Solution:** Use a different port
```bash
# Binary
./whatsapp rest --port 8080

# Docker
docker run -d --name whatsapp --publish 8080:3000 ... rest
```

### Cannot Access Web Interface

**Problem:** Browser shows "Connection refused"

**Solution:**
```bash
# Check if application is running
# Docker
docker ps | grep whatsapp

# Binary
ps aux | grep whatsapp

# Check logs
# Docker
docker logs whatsapp

# Binary (if running with output redirection)
tail -f logs/app.log
```

### QR Code Not Displaying

**Problem:** QR code doesn't appear or expires quickly

**Solution:**
1. Refresh the page
2. Try the pairing code method instead
3. Ensure WhatsApp app is up to date
4. Check internet connection

### Message Not Sending

**Problem:** Message fails to send

**Solution:**
```bash
# Verify phone number format (include country code)
# Correct: 5511999998888
# Wrong: 11999998888

# Check if number is registered on WhatsApp
curl "http://localhost:3000/user/check?phone=5511999998888"
```

## Next Steps

Now that you have the basic setup working, explore these guides:

1. **[Configuration Basics](configuration-basics.md)** - Learn essential configuration options
2. **[First Message Guide](first-message.md)** - Detailed guide for sending different message types
3. **[Installation Guide](installation.md)** - Complete installation options and deployment
4. **[API Documentation](../reference/api/openapi.yaml)** - Full REST API reference
5. **[Webhook Integration](../guides/webhooks/setup.md)** - Receive WhatsApp events

## Basic Configuration

### Add Authentication

**Binary:**
```bash
./whatsapp rest --basic-auth="admin:secret123"
```

**Docker:**
```bash
docker run -d --name whatsapp --publish 3000:3000 \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest \
  --basic-auth="admin:secret123"
```

**Access with authentication:**
```bash
curl -u admin:secret123 http://localhost:3000/app/devices
```

### Enable Debug Logging

**Binary:**
```bash
./whatsapp rest --debug=true
```

**Docker:**
```bash
docker run -d --name whatsapp --publish 3000:3000 \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest \
  --debug=true
```

### Set Custom Device Name

**Binary:**
```bash
./whatsapp rest --os="MyApp"
```

**Docker:**
```bash
docker run -d --name whatsapp --publish 3000:3000 \
  ghcr.io/chatwoot-br/go-whatsapp-web-multidevice rest \
  --os="MyApp"
```

## Testing Your Setup

### Test Message Sending

```bash
# Send text message
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999998888",
    "message": "Test message from API"
  }'

# Expected response
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "message_id": "3EB0C431D4D2E2D2F3E8",
    "status": "sent"
  }
}
```

### Test User Info

```bash
# Get your user info
curl http://localhost:3000/user/info

# Expected response
{
  "code": "SUCCESS",
  "message": "Success",
  "results": {
    "phone": "5511999998888",
    "name": "Your Name",
    "connected": true
  }
}
```

### Test Connection Status

```bash
# Check device connection
curl http://localhost:3000/app/devices

# Expected response
{
  "code": "SUCCESS",
  "message": "Success",
  "results": [
    {
      "device": "Chrome (MyApp)",
      "platform": "Chrome",
      "connected": true
    }
  ]
}
```

## Stopping the Application

### Docker

```bash
# Stop container
docker stop whatsapp

# Remove container
docker rm whatsapp

# Remove volume (deletes all data)
docker volume rm whatsapp-data
```

### Binary

```bash
# Stop the application
# Press Ctrl+C in terminal where it's running

# Or find and kill process
pkill whatsapp
```

## Quick Reference

### Essential Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/app/login` | GET | Get QR code for login |
| `/app/login-with-code?phone=` | GET | Get pairing code |
| `/app/devices` | GET | List connected devices |
| `/app/logout` | GET | Logout from WhatsApp |
| `/user/info` | GET | Get your user info |
| `/send/message` | POST | Send text message |
| `/send/image` | POST | Send image |
| `/user/check?phone=` | GET | Check if number is on WhatsApp |

### Phone Number Format

Always use international format without `+` sign:
- **Brazil**: `5511999998888` (55 = country, 11 = area, 999998888 = number)
- **USA**: `14155552671` (1 = country, 415 = area, 5552671 = number)
- **India**: `919876543210` (91 = country, 9876543210 = number)

### Docker Commands

```bash
# Start container
docker start whatsapp

# Stop container
docker stop whatsapp

# View logs
docker logs -f whatsapp

# Restart container
docker restart whatsapp

# Execute command inside container
docker exec -it whatsapp /bin/sh
```

## Support and Resources

- **GitHub Repository**: [go-whatsapp-web-multidevice](https://github.com/chatwoot-br/go-whatsapp-web-multidevice)
- **Issues**: [GitHub Issues](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/issues)
- **API Documentation**: [OpenAPI Specification](../reference/api/openapi.yaml)
- **Webhook Guide**: [Webhook Documentation](../guides/webhooks/setup.md)

---

**Version**: Compatible with v7.10.0+
**Last Updated**: 2025-12-05
