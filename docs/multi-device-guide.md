# Multi-Device Support Guide

> **Version**: v8.x (introduced in v8.0.0)
> **Last Updated**: December 2024

This guide explains the multi-device support feature in go-whatsapp-web-multidevice, which allows you to connect and manage multiple WhatsApp accounts simultaneously in a single server instance.

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Getting Started](#getting-started)
4. [Device Management API](#device-management-api)
5. [Device Scoping](#device-scoping)
6. [Storage Architecture](#storage-architecture)
7. [Session Management](#session-management)
8. [Webhooks with Multi-Device](#webhooks-with-multi-device)
9. [WebSocket Integration](#websocket-integration)
10. [Backward Compatibility](#backward-compatibility)
11. [Best Practices](#best-practices)
12. [Troubleshooting](#troubleshooting)

---

## Overview

Starting with v8.0.0, go-whatsapp-web-multidevice supports connecting multiple WhatsApp accounts to a single server instance. Each device operates independently with:

- **Isolated sessions**: Each WhatsApp account has its own authentication and connection state
- **Scoped storage**: Chat history and messages are stored separately per device
- **Independent webhooks**: Event payloads include device identification
- **Persistent registry**: Device configurations survive server restarts

### Key Benefits

- Run a single server instance for multiple WhatsApp accounts
- Centralized management through unified API
- Efficient resource utilization
- Simplified deployment and maintenance

---

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                      REST API Layer                         │
│   ┌─────────────────────────────────────────────────────┐   │
│   │            Device Middleware                         │   │
│   │   (Resolves X-Device-Id header or device_id param)  │   │
│   └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Device Manager                           │
│   ┌───────────────────────────────────────────────────┐     │
│   │  Device Registry (map[string]*DeviceInstance)     │     │
│   │                                                   │     │
│   │  ┌─────────┐  ┌─────────┐  ┌─────────┐           │     │
│   │  │Device A │  │Device B │  │Device C │  ...      │     │
│   │  └─────────┘  └─────────┘  └─────────┘           │     │
│   └───────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Per-Device Components                      │
│                                                             │
│  ┌────────────────┐  ┌────────────────┐  ┌──────────────┐  │
│  │ WhatsApp Client│  │  Chat Storage  │  │ Event Handler│  │
│  │   (whatsmeow)  │  │   (scoped)     │  │   (scoped)   │  │
│  └────────────────┘  └────────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Component Details

| Component | Description | Location |
|-----------|-------------|----------|
| **DeviceManager** | Global singleton managing all device instances | `src/infrastructure/whatsapp/device_manager.go` |
| **DeviceInstance** | Per-device wrapper combining client, storage, and metadata | `src/infrastructure/whatsapp/device_instance.go` |
| **DeviceMiddleware** | HTTP middleware resolving device from request | `src/ui/rest/middleware/device.go` |
| **DeviceRepository** | Storage wrapper injecting device_id into all queries | `src/infrastructure/chatstorage/device_repository.go` |

---

## Getting Started

### Starting the Server

```bash
# REST API mode (default port 3000)
./whatsapp rest

# With custom configuration
./whatsapp rest --port=8080 --debug=true
```

### Creating Your First Device

```bash
# Create a new device (generates UUID if no ID provided)
curl -X POST http://localhost:3000/devices

# Create device with custom ID
curl -X POST http://localhost:3000/devices \
  -H "Content-Type: application/json" \
  -d '{"device_id": "my-business-account"}'
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Device created successfully",
  "result": {
    "device_id": "my-business-account",
    "state": "disconnected",
    "created_at": "2024-12-28T10:00:00Z"
  }
}
```

### Logging In

```bash
# QR Code login
curl http://localhost:3000/devices/my-business-account/login

# Pairing code login
curl -X POST http://localhost:3000/devices/my-business-account/login/code \
  -H "Content-Type: application/json" \
  -d '{"phone": "628123456789"}'
```

---

## Device Management API

### Endpoints Overview

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/devices` | List all registered devices |
| `POST` | `/devices` | Create a new device |
| `GET` | `/devices/{device_id}` | Get device details |
| `DELETE` | `/devices/{device_id}` | Remove device from registry |
| `GET` | `/devices/{device_id}/login` | Initiate QR code login |
| `POST` | `/devices/{device_id}/login/code` | Pairing code login |
| `POST` | `/devices/{device_id}/logout` | Logout and purge device |
| `POST` | `/devices/{device_id}/reconnect` | Reconnect to WhatsApp |
| `GET` | `/devices/{device_id}/status` | Get connection status |

### List All Devices

```bash
curl http://localhost:3000/devices
```

**Response:**
```json
{
  "code": "SUCCESS",
  "message": "Devices retrieved successfully",
  "result": [
    {
      "device_id": "personal-account",
      "jid": "628123456789@s.whatsapp.net",
      "display_name": "John Doe",
      "state": "logged_in",
      "created_at": "2024-12-01T10:00:00Z"
    },
    {
      "device_id": "business-account",
      "jid": "628987654321@s.whatsapp.net",
      "display_name": "My Business",
      "state": "connected",
      "created_at": "2024-12-15T14:30:00Z"
    }
  ]
}
```

### Device States

| State | Description |
|-------|-------------|
| `disconnected` | Device registered but not connected |
| `connecting` | Connection in progress |
| `connected` | Connected but not fully logged in |
| `logged_in` | Fully authenticated and ready |

### Logout and Cleanup

```bash
# Logout device (full cleanup: logout, delete storage, remove from registry)
curl -X POST http://localhost:3000/devices/my-business-account/logout
```

This performs a complete purge:
1. Logs out from WhatsApp
2. Deletes device-specific chat storage
3. Removes from device registry
4. Cleans up in-memory instance

---

## Device Scoping

All device-scoped operations require device identification. There are two ways to specify the device:

### Method 1: HTTP Header (Recommended)

```bash
curl http://localhost:3000/send/message \
  -H "X-Device-Id: my-business-account" \
  -H "Content-Type: application/json" \
  -d '{"phone": "628123456789", "message": "Hello!"}'
```

### Method 2: Query Parameter

```bash
curl "http://localhost:3000/send/message?device_id=my-business-account" \
  -H "Content-Type: application/json" \
  -d '{"phone": "628123456789", "message": "Hello!"}'
```

### Single-Device Fallback

If only one device is registered, you can omit the device identifier. The system automatically uses the default device:

```bash
# Works when only one device is registered
curl http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{"phone": "628123456789", "message": "Hello!"}'
```

### Device-Scoped Endpoints

These endpoints require device identification:

| Category | Endpoints |
|----------|-----------|
| **Messaging** | `/send/message`, `/send/image`, `/send/file`, `/send/video`, `/send/audio`, `/send/sticker`, `/send/contact`, `/send/link`, `/send/location`, `/send/poll`, `/send/presence` |
| **User** | `/user/info`, `/user/avatar`, `/user/my/groups`, `/user/my/contacts`, `/user/check` |
| **Chat** | `/chats`, `/chat/{jid}/messages`, `/chat/{jid}/archive`, `/chat/{jid}/pin`, `/chat/{jid}/label` |
| **Message** | `/message/{id}/revoke`, `/message/{id}/reaction`, `/message/{id}/delete`, `/message/{id}/read`, `/message/{id}/download` |
| **Group** | `/group/*` (all group endpoints) |
| **Newsletter** | `/newsletter/*` |

---

## Storage Architecture

### Database Layout

Multi-device support uses a three-database model:

```
storages/
├── whatsapp.db        # Primary WhatsApp store (auth, sessions)
├── whatsapp_keys.db   # Encryption keys (optional, can share with primary)
└── chatstorage.db     # Chat history with device isolation
```

### Chat Storage Schema

```sql
-- Device registry table
CREATE TABLE devices (
    device_id VARCHAR(255) PRIMARY KEY,
    display_name VARCHAR(255),
    jid VARCHAR(255),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Chats table (device-scoped)
CREATE TABLE chats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id VARCHAR(255) NOT NULL,
    jid VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    unread_count INTEGER DEFAULT 0,
    last_message_at TIMESTAMP,
    created_at TIMESTAMP,
    UNIQUE(device_id, jid)
);

-- Messages table (device-scoped)
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    chat_jid VARCHAR(255) NOT NULL,
    sender VARCHAR(255),
    content TEXT,
    timestamp TIMESTAMP,
    is_from_me BOOLEAN,
    UNIQUE(device_id, message_id)
);
```

### Storage Isolation

All storage operations are automatically scoped to the current device:

```go
// Internal implementation - DeviceRepository wraps all queries
func (r *DeviceRepository) SaveMessage(ctx context.Context, msg Message) error {
    msg.DeviceID = r.deviceID  // Automatically injected
    return r.base.SaveMessage(ctx, msg)
}
```

---

## Session Management

### Session Persistence

Device sessions are persisted across server restarts:

1. **WhatsApp Authentication**: Stored in primary database (whatsapp.db)
2. **Device Registry**: Stored in chat storage database
3. **Lazy Loading**: Devices are loaded on startup but clients connect on-demand

### Startup Flow

```
Server Start
    │
    ▼
Load Configuration
    │
    ▼
Initialize Databases
    │
    ▼
Create DeviceManager
    │
    ▼
LoadExistingDevices() ─── Load device registry from storage
    │
    ▼
Start REST Server
    │
    ▼
Start Auto-Reconnect Checker ─── Background goroutine
    │
    ▼
Ready for Requests
```

### Auto-Reconnect

The server includes an auto-reconnect feature that monitors device connections:

```go
// Background checker runs periodically
func startAutoReconnectChecker() {
    for device := range deviceManager.ListDevices() {
        if device.IsDisconnected() && device.WasLoggedIn() {
            device.Reconnect()
        }
    }
}
```

### Connection Lifecycle

```
1. Create Device (POST /devices)
   └─ Device registered as placeholder (no client)

2. Login (GET /devices/{id}/login)
   └─ EnsureClient() creates WhatsApp client
   └─ QR code or pairing code generated

3. Scan/Pair
   └─ Client authenticates with WhatsApp
   └─ State changes to "logged_in"

4. Send/Receive Messages
   └─ All operations use device-scoped client
   └─ Events broadcast to webhooks/WebSocket

5. Disconnect (network issue)
   └─ Auto-reconnect attempts in background

6. Logout (POST /devices/{id}/logout)
   └─ Full cleanup and removal
```

---

## Webhooks with Multi-Device

### Payload Format

All webhook payloads now include a top-level `device_id` field:

```json
{
  "event": "message",
  "device_id": "628123456789@s.whatsapp.net",
  "payload": {
    "message_id": "3EB0A1B2C3D4E5F6",
    "chat_jid": "628987654321@s.whatsapp.net",
    "sender": "628987654321@s.whatsapp.net",
    "content": "Hello, how can I help?",
    "timestamp": "2024-12-28T15:30:00Z",
    "is_from_me": false
  }
}
```

### Configuration

```bash
# Multiple webhooks (comma-separated)
./whatsapp rest --webhook="https://webhook1.com,https://webhook2.com"

# Event filtering
./whatsapp rest --webhook-events="message,message.ack,group.participants"

# Security options
./whatsapp rest \
  --webhook-secret="your-secret-key" \
  --webhook-insecure-skip-verify=false
```

### Event Types

| Event | Description |
|-------|-------------|
| `message` | Text, media, contact, location messages |
| `message.reaction` | Emoji reactions to messages |
| `message.revoked` | Deleted/revoked messages |
| `message.edited` | Edited messages |
| `message.ack` | Delivery and read receipts |
| `group.participants` | Group member join/leave/promote/demote |

### Identifying Device in Webhook Handler

```javascript
// Node.js webhook handler example
app.post('/webhook', (req, res) => {
  const { event, device_id, payload } = req.body;

  console.log(`Event ${event} from device ${device_id}`);

  // Route to appropriate handler based on device
  switch (device_id) {
    case 'support-account':
      handleSupportMessage(payload);
      break;
    case 'sales-account':
      handleSalesMessage(payload);
      break;
  }

  res.status(200).send('OK');
});
```

---

## WebSocket Integration

### Device-Scoped WebSocket

Connect to WebSocket with device scope:

```javascript
// Connect to specific device's WebSocket
const ws = new WebSocket('ws://localhost:3000/ws?device_id=my-business-account');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`Event: ${data.code}`, data.result);
};
```

### Broadcast Events

```json
{
  "code": "MESSAGE_RECEIVED",
  "message": "New message received",
  "result": {
    "device_id": "my-business-account",
    "message": {
      "id": "3EB0A1B2C3D4E5F6",
      "from": "628123456789@s.whatsapp.net",
      "content": "Hello!"
    }
  }
}
```

---

## Backward Compatibility

### Single-Device Mode

For backward compatibility, if only one device is registered:

- Device identifier is optional in API calls
- System automatically uses the only registered device
- Legacy `/app/*` endpoints still work

### Legacy Endpoints

These endpoints remain available for single-device usage:

| Legacy | New Multi-Device |
|--------|------------------|
| `GET /app/login` | `GET /devices/{id}/login` |
| `POST /app/login-with-code` | `POST /devices/{id}/login/code` |
| `POST /app/logout` | `POST /devices/{id}/logout` |
| `POST /app/reconnect` | `POST /devices/{id}/reconnect` |
| `GET /app/status` | `GET /devices/{id}/status` |
| `GET /app/devices` | `GET /devices` |

---

## Best Practices

### Device Naming

Use descriptive, consistent device IDs:

```bash
# Good examples
my-business-account
support-team-1
sales-brazil
notifications-bot

# Avoid
device1
abc123
temp
```

### Resource Management

1. **Limit concurrent devices**: Each device maintains a WebSocket connection to WhatsApp
2. **Monitor connection states**: Use `/devices/{id}/status` to check health
3. **Clean up unused devices**: Remove devices that are no longer needed

### Error Handling

```bash
# Check device status before operations
status=$(curl -s http://localhost:3000/devices/my-account/status | jq -r '.result.state')
if [ "$status" != "logged_in" ]; then
  echo "Device not ready: $status"
  exit 1
fi
```

### Production Deployment

```yaml
# docker-compose.yml for multi-device
services:
  whatsapp:
    image: ghcr.io/aldinokemal/go-whatsapp-web-multidevice
    container_name: whatsapp-multidevice
    restart: always
    ports:
      - "3000:3000"
    volumes:
      - whatsapp_data:/app/storages
    environment:
      - APP_DEBUG=false
      - WHATSAPP_WEBHOOK=https://your-webhook.com/handler
      - WHATSAPP_WEBHOOK_SECRET=your-secret-key

volumes:
  whatsapp_data:
```

---

## Troubleshooting

### Common Issues

#### Device Not Found

```json
{
  "code": "DEVICE_NOT_FOUND",
  "message": "Device 'my-account' not found"
}
```

**Solution**: Check device exists with `GET /devices` and ensure correct device_id.

#### Multiple Devices Without Identifier

```json
{
  "code": "DEVICE_ID_REQUIRED",
  "message": "Multiple devices registered, device_id is required"
}
```

**Solution**: Provide `X-Device-Id` header or `device_id` query parameter.

#### Device Not Logged In

```json
{
  "code": "DEVICE_NOT_LOGGED_IN",
  "message": "Device 'my-account' is not logged in"
}
```

**Solution**: Login device via `/devices/{id}/login` or reconnect via `/devices/{id}/reconnect`.

### Debug Mode

Enable debug logging for troubleshooting:

```bash
./whatsapp rest --debug=true
```

### Health Check Script

```bash
#!/bin/bash
# health-check.sh

API_URL="http://localhost:3000"

# List all devices and check status
devices=$(curl -s "$API_URL/devices" | jq -r '.result[].device_id')

for device in $devices; do
  status=$(curl -s "$API_URL/devices/$device/status" | jq -r '.result.state')
  echo "Device: $device - Status: $status"

  if [ "$status" == "disconnected" ]; then
    echo "  Attempting reconnect..."
    curl -s -X POST "$API_URL/devices/$device/reconnect"
  fi
done
```

---

## API Reference

For complete API documentation, see:

- [OpenAPI Specification](./openapi.yaml)
- [Webhook Payload Documentation](./webhook-payload.md)
- [Online API Documentation](https://bump.sh/aldinokemal/doc/go-whatsapp-web-multidevice)

---

## Version History

| Version | Changes |
|---------|---------|
| v8.0.0 | Initial multi-device support |
| v8.1.0 | Device persistence, auto-reconnect improvements |

---

## Support

- **Issues**: [GitHub Issues](https://github.com/aldinokemal/go-whatsapp-web-multidevice/issues)
- **Patreon**: [Support Development](https://www.patreon.com/c/aldinokemal)
