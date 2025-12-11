# Documentation

Welcome to the go-whatsapp-web-multidevice documentation! This guide will help you find the information you need quickly.

## Quick Navigation by Role

### 👤 I'm New Here

Start here if you're new to the project:

1. **[Quick Start](getting-started/quick-start.md)** - Get up and running in 5 minutes
2. **[Installation Guide](getting-started/installation.md)** - Detailed installation instructions
3. **[Send Your First Message](getting-started/first-message.md)** - Basic usage example
4. **[Configuration Basics](getting-started/configuration-basics.md)** - Essential settings

### 🚀 I Want to Deploy

Choose your deployment method:

- **[Docker Deployment](guides/deployment/docker.md)** - Containerized deployment
- **[Kubernetes](guides/deployment/kubernetes.md)** - Deploy with Helm charts
- **[Binary Deployment](guides/deployment/binary.md)** - Standalone binary with systemd
- **[Production Checklist](guides/deployment/production-checklist.md)** - Pre-deployment requirements

### 📱 I Want to Integrate

Integration guides for common tasks:

- **[Webhooks Setup](guides/webhooks/setup.md)** - Receive real-time events
- **[Webhook Security](guides/webhooks/security.md)** - HMAC signature verification
- **[Webhook Examples](guides/webhooks/examples.md)** - Working integration code
- **[MCP Integration](guides/mcp-integration.md)** - AI agent integration
- **[Admin API](guides/admin-api.md)** - Multi-instance management
- **[Media Handling](guides/media-handling.md)** - Send and process media

### 📖 I Need API Reference

Technical specifications and schemas:

- **[OpenAPI Specification](reference/api/openapi.yaml)** - Complete REST API spec
- **[API Guide](reference/api/openapi.md)** - Human-readable API documentation
- **[Admin API Spec](reference/api/admin-api-openapi.yaml)** - Admin API specification
- **[Webhook Event Types](reference/webhooks/event-types.md)** - All webhook events
- **[Webhook Payload Schemas](reference/webhooks/payload-schemas.md)** - Detailed payloads
- **[Configuration Reference](reference/configuration.md)** - All environment variables
- **[Phone Number Format](reference/phone-number-format.md)** - JID format explained
- **[Troubleshooting](reference/troubleshooting.md)** - Common issues

### 👨‍💻 I Want to Contribute

Developer resources:

- **[Architecture Overview](developer/architecture.md)** - System design and patterns
- **[Contributing Guide](developer/contributing.md)** - How to contribute
- **[Documentation Guide](developer/documentation-guide.md)** - Maintaining documentation
- **[Testing Guide](developer/testing.md)** - Writing and running tests
- **[Release Process](developer/release-process.md)** - Creating new releases
- **[ADR-0001: Admin API](developer/adr/0001-admin-api.md)** - Architecture decisions

### ⚙️ I'm Running in Production

Operations and maintenance:

- **[Monitoring Guide](operations/monitoring.md)** - Metrics and logging
- **[Performance Tuning](operations/performance-tuning.md)** - Optimization strategies
- **[Security Best Practices](operations/security-best-practices.md)** - Security guidelines
- **[Audio Optimization](operations/audio-optimization.md)** - FFmpeg configuration

### 🔍 I'm Investigating an Issue

Troubleshooting and lessons learned:

- **[Troubleshooting Guide](reference/troubleshooting.md)** - Common issues and solutions
- **[Postmortem: Profile Picture Panic](postmortems/001-profile-picture-panic.md)** - Service crash analysis
- **[Postmortem: Encryption Failure](postmortems/002-multidevice-encryption.md)** - Multi-device issues
- **[Postmortem: Auto-Reconnect Panic](postmortems/003-auto-reconnect-panic.md)** - Reconnection crash
- **[Postmortem: Media Filename Issue](postmortems/004-media-filename-mime-pollution.md)** - MIME type pollution

---

## Documentation Structure

Our documentation follows the [Divio documentation system](https://documentation.divio.com/):

```
docs/
├── getting-started/     📚 Learning-oriented tutorials
│   ├── quick-start.md
│   ├── installation.md
│   ├── first-message.md
│   └── configuration-basics.md
│
├── guides/              🎯 Problem-oriented how-to guides
│   ├── deployment/
│   │   ├── docker.md
│   │   ├── kubernetes.md
│   │   ├── binary.md
│   │   └── production-checklist.md
│   ├── webhooks/
│   │   ├── setup.md
│   │   ├── security.md
│   │   └── examples.md
│   ├── mcp-integration.md
│   ├── admin-api.md
│   └── media-handling.md
│
├── reference/           📖 Information-oriented reference material
│   ├── api/
│   │   ├── openapi.yaml
│   │   ├── openapi.md
│   │   └── admin-api-openapi.yaml
│   ├── webhooks/
│   │   ├── event-types.md
│   │   └── payload-schemas.md
│   ├── configuration.md
│   ├── phone-number-format.md
│   └── troubleshooting.md
│
├── developer/           🔧 Understanding-oriented explanations
│   ├── architecture.md
│   ├── contributing.md
│   ├── testing.md
│   ├── release-process.md
│   └── adr/
│       └── 0001-admin-api.md
│
├── operations/          ⚙️ Production operations
│   ├── monitoring.md
│   ├── performance-tuning.md
│   ├── security-best-practices.md
│   └── audio-optimization.md
│
├── postmortems/         📝 Lessons learned
│   ├── 001-profile-picture-panic.md
│   ├── 002-multidevice-encryption.md
│   ├── 003-auto-reconnect-panic.md
│   ├── 004-media-filename-mime-pollution.md
│   └── lessons-learned.md
│
└── plans/               📋 Implementation plans
    └── 0001-fix-code-review-issues.md
```

## Common Tasks

### Getting Started

```bash
# 1. Clone the repository
git clone https://github.com/chatwoot-br/go-whatsapp-web-multidevice
cd go-whatsapp-web-multidevice

# 2. Build the binary
cd src && go build -o whatsapp

# 3. Run REST API mode
./whatsapp rest

# 4. Run MCP mode
./whatsapp mcp
```

See [Quick Start Guide](getting-started/quick-start.md) for details.

### Sending a Message

```bash
curl -X POST http://localhost:3000/send/text \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "6281234567890",
    "message": "Hello from WhatsApp API!"
  }'
```

See [First Message Guide](getting-started/first-message.md) for more examples.

### Setting Up Webhooks

```bash
# Set webhook URL in .env
WHATSAPP_WEBHOOK=https://your-webhook.com/handler
WHATSAPP_WEBHOOK_SECRET=your-secret-key

# Start the server
./whatsapp rest
```

See [Webhooks Setup Guide](guides/webhooks/setup.md) for details.

## Key Features

- **Multi-Device Support**: WhatsApp Web multi-device protocol
- **REST API**: Complete HTTP API for all WhatsApp operations
- **MCP Server**: Model Context Protocol for AI agent integration
- **Webhooks**: Real-time event notifications
- **Media Support**: Images, videos, audio, documents, stickers
- **Group Management**: Create, manage, and interact with groups
- **Admin API**: Manage multiple WhatsApp instances
- **Auto-Reconnect**: Automatic reconnection on disconnects
- **Chat Storage**: Optional message history storage

## Architecture

```
┌────────────────────────────────────────┐
│        Entry Point (CLI)               │
│     Cobra + Viper Framework            │
└──────────────┬─────────────────────────┘
               │
        ┌──────┴──────┐
        │             │
        ▼             ▼
    ┌──────┐     ┌──────┐
    │ REST │     │ MCP  │
    │ Mode │     │ Mode │
    └──┬───┘     └──┬───┘
       │            │
       └─────┬──────┘
             │
    ┌────────▼────────┐
    │   UI Layer      │
    └────────┬────────┘
             │
    ┌────────▼────────┐
    │  Usecase Layer  │
    └────────┬────────┘
             │
    ┌────────▼────────┐
    │  Domain Layer   │
    └────────┬────────┘
             │
    ┌────────▼────────────┐
    │ Infrastructure      │
    │ (WhatsApp/Database) │
    └─────────────────────┘
```

See [Architecture Overview](developer/architecture.md) for detailed explanation.

## Configuration

Key environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_PORT` | `3000` | HTTP server port |
| `APP_DEBUG` | `false` | Enable debug logging |
| `APP_BASIC_AUTH` | - | Basic auth credentials |
| `DB_URI` | `file:storages/whatsapp.db` | Database connection |
| `WHATSAPP_WEBHOOK` | - | Webhook URLs |
| `WHATSAPP_WEBHOOK_SECRET` | `secret` | HMAC secret |

See [Configuration Reference](reference/configuration.md) for complete list.

## API Modes

### REST API Mode

- HTTP server with web UI
- REST API endpoints
- WebSocket for real-time updates
- Default port: 3000

```bash
./whatsapp rest --port 3000
```

### MCP Server Mode

- Model Context Protocol server
- SSE (Server-Sent Events) transport
- AI agent integration
- Default port: 8080

```bash
./whatsapp mcp --port 8080
```

**Note**: Only one mode can run at a time.

## Support

- **Documentation**: You're reading it!
- **Issues**: [GitHub Issues](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/issues)
- **Discussions**: [GitHub Discussions](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/discussions)
- **Releases**: [GitHub Releases](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/releases)

## Contributing

We welcome contributions! See our [Contributing Guide](developer/contributing.md) for:

- Development setup
- Coding standards
- Testing requirements
- Pull request process

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Related Projects

- [whatsmeow](https://github.com/tulir/whatsmeow) - WhatsApp Web multi-device protocol library
- [Fiber](https://github.com/gofiber/fiber) - Web framework
- [mcp-go](https://github.com/mark3labs/mcp-go) - Model Context Protocol library

## Version

Current version: v7.10.1

See [CHANGELOG](../CHANGELOG.md) for version history.

---

**Need help?** Check the [Troubleshooting Guide](reference/troubleshooting.md) or [open an issue](https://github.com/chatwoot-br/go-whatsapp-web-multidevice/issues).
