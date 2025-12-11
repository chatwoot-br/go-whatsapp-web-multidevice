# MCP Integration Guide

The Model Context Protocol (MCP) allows AI agents and tools to interact with WhatsApp through a standardized protocol. This guide explains how to set up and use the WhatsApp MCP server.

## Table of Contents

- [What is MCP?](#what-is-mcp)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Available Tools](#available-tools)
- [Integration Examples](#integration-examples)
- [Troubleshooting](#troubleshooting)

## What is MCP?

**Model Context Protocol (MCP)** is a standardized protocol for AI agents to interact with external services and tools. It provides:

- **Standardized Communication**: Consistent interface for AI agents
- **Server-Sent Events (SSE)**: Real-time communication via HTTP
- **Tool Discovery**: AI agents can discover available capabilities
- **Resource Access**: Structured access to data and operations

### Why Use MCP Mode?

- **AI Integration**: Perfect for AI assistants (Claude Desktop, Cursor, etc.)
- **Standardized Protocol**: Works with any MCP-compatible client
- **Real-time Communication**: SSE transport for efficient updates
- **Tool-based Interface**: AI agents can discover and use WhatsApp operations

### MCP vs REST Mode

| Feature | MCP Mode | REST Mode |
|---------|----------|-----------|
| **Interface** | Tool-based (for AI) | HTTP endpoints |
| **Transport** | Server-Sent Events | HTTP Request/Response |
| **Use Case** | AI agent integration | Traditional API clients |
| **Web UI** | No | Yes |
| **Documentation** | Self-describing tools | OpenAPI spec |

**Note**: Only one mode can run at a time due to WhatsApp client limitations.

## Quick Start

### Prerequisites

- Go 1.22 or later
- FFmpeg (for media processing)
- WhatsApp account with multi-device support

### Installation

1. **Clone the repository**:

```bash
git clone https://github.com/chatwoot-br/go-whatsapp-web-multidevice
cd go-whatsapp-web-multidevice
```

2. **Build the binary**:

```bash
cd src
go build -o whatsapp
```

3. **Start the MCP server**:

```bash
./whatsapp mcp
```

The server will start on `http://localhost:8080` by default.

### Server Options

Customize the MCP server with command-line flags:

```bash
./whatsapp mcp --host localhost --port 8080
```

**Available Flags**:

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `localhost` | Host address for the MCP server |
| `--port` | `8080` | Port for the MCP server |
| `--debug` | `false` | Enable debug logging |

### Verify Server is Running

The MCP server exposes two endpoints:

- **SSE endpoint**: `http://localhost:8080/sse` (for client connections)
- **Message endpoint**: `http://localhost:8080/message` (for sending messages)

Check if the server is running:

```bash
curl http://localhost:8080/sse
```

## Configuration

### Environment Variables

The MCP server uses the same configuration as REST mode. Create a `.env` file in the `src/` directory:

```bash
# Database
DB_URI=file:storages/whatsapp.db
DB_KEYS_URI=file::memory:

# WhatsApp
WHATSAPP_AUTO_REPLY=
WHATSAPP_AUTO_MARK_READ=false
WHATSAPP_WEBHOOK=
WHATSAPP_WEBHOOK_SECRET=secret
WHATSAPP_ACCOUNT_VALIDATION=true

# Application
APP_DEBUG=false
APP_OS=Chrome
```

See [Configuration Reference](../reference/configuration.md) for all options.

### MCP Client Configuration

#### Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "whatsapp": {
      "url": "http://localhost:8080/sse"
    }
  }
}
```

#### Cursor

Add to your Cursor MCP settings:

```json
{
  "mcpServers": {
    "whatsapp": {
      "url": "http://localhost:8080/sse"
    }
  }
}
```

#### Generic MCP Client

For any MCP-compatible client supporting SSE transport:

```json
{
  "servers": {
    "whatsapp": {
      "transport": "sse",
      "url": "http://localhost:8080/sse"
    }
  }
}
```

## Available Tools

The WhatsApp MCP server provides comprehensive tools for AI agents to interact with WhatsApp. Tools are automatically discovered by MCP clients.

### 📱 Connection Management

| Tool | Description |
|------|-------------|
| `whatsapp_connection_status` | Check whether the WhatsApp client is connected and logged in |
| `whatsapp_login_qr` | Initiate QR code based login flow with image output |
| `whatsapp_login_with_code` | Generate pairing code for multi-device login using phone number |
| `whatsapp_logout` | Sign out the current WhatsApp session |
| `whatsapp_reconnect` | Attempt to reconnect to WhatsApp using stored session |

### 💬 Messaging & Communication

| Tool | Description |
|------|-------------|
| `whatsapp_send_text` | Send text messages with reply and forwarding support |
| `whatsapp_send_contact` | Send contact cards with name and phone number |
| `whatsapp_send_link` | Send links with custom captions |
| `whatsapp_send_location` | Send location coordinates (latitude/longitude) |
| `whatsapp_send_image` | Send images with captions, compression, and view-once options |
| `whatsapp_send_sticker` | Send stickers with automatic WebP conversion (JPG/PNG/GIF) |

### 📋 Chat & Contact Management

| Tool | Description |
|------|-------------|
| `whatsapp_list_contacts` | Retrieve all contacts in your WhatsApp account |
| `whatsapp_list_chats` | Get recent chats with pagination and search filters |
| `whatsapp_get_chat_messages` | Fetch messages from specific chats with time/media filtering |
| `whatsapp_download_message_media` | Download images/videos from messages |

### 👥 Group Management

| Tool | Description |
|------|-------------|
| `whatsapp_group_create` | Create new groups with optional initial participants |
| `whatsapp_group_join_via_link` | Join groups using invite links |
| `whatsapp_group_leave` | Leave groups by group ID |
| `whatsapp_group_participants` | List all participants in a group |
| `whatsapp_group_manage_participants` | Add, remove, promote, or demote group members |
| `whatsapp_group_invite_link` | Get or reset group invite links |
| `whatsapp_group_info` | Get detailed group information |
| `whatsapp_group_set_name` | Update group display name |
| `whatsapp_group_set_topic` | Update group description/topic |
| `whatsapp_group_set_locked` | Toggle admin-only group info editing |
| `whatsapp_group_set_announce` | Toggle announcement-only mode |
| `whatsapp_group_join_requests` | List pending join requests |
| `whatsapp_group_manage_join_requests` | Approve or reject join requests |

### Tool Categories Summary

- **Connection**: 5 tools for authentication and connection management
- **Messaging**: 6 tools for sending messages and media
- **Chat/Contact**: 4 tools for querying chats and contacts
- **Group**: 13 tools for comprehensive group management

## Integration Examples

### Example 1: Login with QR Code

When using an MCP client (like Claude Desktop), you can authenticate:

```
User: Login to WhatsApp using QR code

AI: I'll initiate the QR code login.
[AI calls whatsapp_login_qr tool]
[QR code image is displayed]

Scan this QR code with your WhatsApp mobile app:
1. Open WhatsApp on your phone
2. Go to Settings > Linked Devices
3. Tap "Link a Device"
4. Scan the QR code
```

### Example 2: Send a Text Message

```
User: Send a message to John saying "Meeting at 3pm"

AI: I'll send that message to John.
[AI calls whatsapp_send_text tool]

Parameters:
- phone: "6281234567890@s.whatsapp.net"
- message: "Meeting at 3pm"

Message sent successfully!
```

### Example 3: List Recent Chats

```
User: Show me my recent WhatsApp chats

AI: I'll fetch your recent chats.
[AI calls whatsapp_list_chats tool]

Here are your recent chats:
1. John Doe (last message: 2 hours ago)
2. Work Group (last message: 1 day ago)
3. Family Group (last message: 3 days ago)
```

### Example 4: Create a Group

```
User: Create a group called "Project Team" with Alice and Bob

AI: I'll create the group with those participants.
[AI calls whatsapp_group_create tool]

Parameters:
- name: "Project Team"
- participants: ["6281234567890@s.whatsapp.net", "6289876543210@s.whatsapp.net"]

Group created successfully!
Group ID: 120363123456789012@g.us
```

### Example 5: Download Media

```
User: Download the image from the latest message in John's chat

AI: I'll fetch the messages and download the image.
[AI calls whatsapp_get_chat_messages tool]
[AI identifies message with media]
[AI calls whatsapp_download_message_media tool]

Image downloaded successfully!
File: message_image_20250114.jpg
Size: 2.3 MB
```

## Phone Number Format

WhatsApp uses **JID (Jabber ID)** format for phone numbers:

- **Individual**: `[country_code][phone_number]@s.whatsapp.net`
  - Example: `6281234567890@s.whatsapp.net`
- **Group**: `[group_id]@g.us`
  - Example: `120363123456789012@g.us`

**Important**: No `+`, `-`, or spaces. Just digits + `@s.whatsapp.net`.

## Webhooks with MCP Mode

You can still use webhooks in MCP mode to receive real-time events:

```bash
./whatsapp mcp --port 8080
```

Set webhook in `.env`:

```bash
WHATSAPP_WEBHOOK=https://your-webhook-url.com/webhook
WHATSAPP_WEBHOOK_SECRET=your-secret-key
```

See [Webhooks Guide](webhooks/setup.md) for details.

## Troubleshooting

### Server Won't Start

**Problem**: `Failed to start SSE server: address already in use`

**Solution**: Another process is using port 8080.

```bash
# Check what's using the port
lsof -i :8080

# Use a different port
./whatsapp mcp --port 8081
```

### Connection Issues

**Problem**: AI client can't connect to MCP server

**Solutions**:

1. **Verify server is running**:
   ```bash
   curl http://localhost:8080/sse
   ```

2. **Check firewall**: Ensure port 8080 is not blocked

3. **Try different host**:
   ```bash
   ./whatsapp mcp --host 0.0.0.0 --port 8080
   ```

4. **Check client configuration**: Ensure URL is correct in client config

### WhatsApp Not Logged In

**Problem**: Tools return "not logged in" error

**Solution**: Login first using `whatsapp_login_qr` or `whatsapp_login_with_code`

```
User: Login to WhatsApp
AI: [Uses whatsapp_login_qr tool]
```

### Tools Not Appearing

**Problem**: AI client doesn't show WhatsApp tools

**Solutions**:

1. **Restart the MCP server**
2. **Restart the AI client** (Claude Desktop, Cursor, etc.)
3. **Check client logs** for connection errors
4. **Verify configuration** is correct

### Debug Mode

Enable debug logging to troubleshoot issues:

```bash
./whatsapp mcp --debug=true
```

This will show:
- Incoming tool calls
- WhatsApp protocol events
- Connection status changes
- Error details

## Architecture

The MCP server architecture:

```
┌─────────────────────────────────────┐
│   MCP Client (Claude/Cursor/etc)    │
└────────────┬────────────────────────┘
             │ SSE Connection
             │ (http://localhost:8080/sse)
             ▼
┌─────────────────────────────────────┐
│      MCP Server (mcp-go)            │
│  - Tool Registration                │
│  - Request Routing                  │
│  - Response Formatting              │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│    Tool Handlers (ui/mcp/)          │
│  - SendHandler                      │
│  - QueryHandler                     │
│  - AppHandler                       │
│  - GroupHandler                     │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│      Usecases (usecase/)            │
│  Business Logic Orchestration       │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│      Domains (domains/)             │
│  Core Business Logic                │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│  Infrastructure (infrastructure/)   │
│  - WhatsApp Client (whatsmeow)      │
│  - Database                         │
└─────────────────────────────────────┘
```

See [Architecture Overview](../developer/architecture.md) for details.

## Best Practices

1. **Authentication First**: Always login before using other tools
2. **Phone Number Format**: Use JID format correctly
3. **Error Handling**: AI agents will handle errors gracefully
4. **Rate Limiting**: Be mindful of WhatsApp rate limits
5. **Media Sizes**: Keep media within size limits (images: 20MB, videos: 100MB)
6. **Debug Mode**: Use for development, disable in production

## Limitations

- **Single Mode**: Cannot run REST and MCP mode simultaneously
- **WhatsApp Limits**: Subject to WhatsApp rate limits and policies
- **Media Processing**: Requires FFmpeg for compression
- **Session Management**: One WhatsApp account per server instance

## Security Considerations

- **Local Only**: By default, MCP server binds to localhost
- **No Authentication**: MCP protocol doesn't include built-in auth
- **Production**: For remote access, use a reverse proxy with authentication
- **Webhook Security**: Use HMAC verification for webhooks

Example with nginx:

```nginx
server {
    listen 443 ssl;
    server_name whatsapp-mcp.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    auth_basic "Restricted";
    auth_basic_user_file /etc/nginx/.htpasswd;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

## Deployment

### Docker

```dockerfile
FROM golang:1.22-alpine

WORKDIR /app
COPY src/ .

RUN apk add --no-cache ffmpeg
RUN go build -o whatsapp

CMD ["./whatsapp", "mcp", "--host", "0.0.0.0", "--port", "8080"]
```

```bash
docker build -t whatsapp-mcp .
docker run -p 8080:8080 -v whatsapp-data:/app/storages whatsapp-mcp
```

### Systemd Service

Create `/etc/systemd/system/whatsapp-mcp.service`:

```ini
[Unit]
Description=WhatsApp MCP Server
After=network.target

[Service]
Type=simple
User=whatsapp
WorkingDirectory=/opt/whatsapp
ExecStart=/opt/whatsapp/whatsapp mcp --host localhost --port 8080
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable whatsapp-mcp
sudo systemctl start whatsapp-mcp
sudo systemctl status whatsapp-mcp
```

## Related Documentation

- [Architecture Overview](../developer/architecture.md) - System design
- [API Reference](../reference/api/) - REST API (alternative to MCP)
- [Webhooks Guide](webhooks/setup.md) - Real-time events
- [Configuration Reference](../reference/configuration.md) - All settings
- [Troubleshooting](../reference/troubleshooting.md) - Common issues

## Additional Resources

- [MCP Specification](https://modelcontextprotocol.io/)
- [mcp-go Library](https://github.com/mark3labs/mcp-go)
- [whatsmeow Documentation](https://github.com/tulir/whatsmeow)

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/issues)
- **Discussions**: [GitHub Discussions](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/discussions)
- **Documentation**: Check other guides in `docs/`
