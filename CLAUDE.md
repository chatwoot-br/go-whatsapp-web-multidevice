# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building and Running

- **Build binary**: `cd src && go build -o whatsapp` (Linux/macOS) or `go build -o whatsapp.exe` (Windows)
- **Run REST API mode**: `cd src && go run . rest` or `./whatsapp rest`
- **Run MCP server mode**: `cd src && go run . mcp` or `./whatsapp mcp`
- **Run with Docker**: `docker-compose up -d --build`

### Testing

- **Run all tests**: `cd src && go test ./...`
- **Run specific package tests**: `cd src && go test ./validations`
- **Run tests with coverage**: `cd src && go test -cover ./...`

### Development

- **Format code**: `cd src && go fmt ./...`
- **Get dependencies**: `cd src && go mod tidy`
- **Check for issues**: `cd src && go vet ./...`

### Releasing a New Version

When you need to release a new version (e.g., after fixing bugs or adding features):

1. **Determine version number** (following [Semantic Versioning](https://semver.org/)):
   - **PATCH** (v7.7.1): Bug fixes, backward compatible
   - **MINOR** (v7.8.0): New features, backward compatible
   - **MAJOR** (v8.0.0): Breaking changes

2. **Update version in three files**:
   ```bash
   # 1. Update src/config/settings.go line 8
   # Change: AppVersion = "v7.7.1"

   # 2. Update charts/gowa/Chart.yaml line 18
   # Change: version: 7.7.1

   # 3. Update charts/gowa/Chart.yaml line 24
   # Change: appVersion: "v7.7.1"

   # 4. Update CHANGELOG.md
   # Add new version section at the top
   ```

3. **Commit and tag**:
   ```bash
   git add src/config/settings.go charts/gowa/Chart.yaml CHANGELOG.md
   git commit -m "chore: bump version to v7.7.1"
   git tag -a v7.7.1 -m "Release v7.7.1"
   git push origin main
   git push origin v7.7.1
   ```

4. **Automated builds**: After pushing the tag, GitHub Actions will automatically:
   - Build Docker images for AMD64 and ARM64
   - Push to GitHub Container Registry
   - Create Helm chart release
   - Create GitHub release

5. **Verify release**: Check that all GitHub Actions workflows completed successfully

For detailed release instructions, see [Release Process Documentation](docs/RELEASE-PROCESS.md).

## Project Architecture

This is a Go-based WhatsApp Web API server supporting both REST API and MCP (Model Context Protocol) modes.

### Core Architecture Pattern

- **Domain-Driven Design**: Business logic separated into domain packages (`domains/`)
- **Clean Architecture**: Clear separation between UI, use cases, and infrastructure layers
- **Cobra CLI**: Command pattern with separate commands for `rest` and `mcp` modes

### Key Directories

- `src/`: Main source code directory
- `src/cmd/`: CLI commands (root, rest, mcp)
- `src/domains/`: Business domain logic (app, chat, group, message, send, user, newsletter)
- `src/infrastructure/`: External integrations (WhatsApp, database)
- `src/ui/`: User interface layers (REST API, MCP server, WebSocket)
- `src/usecase/`: Application use cases bridging domains and UI
- `src/validations/`: Input validation logic
- `src/pkg/`: Shared utilities and helpers

### Configuration

- **Environment Variables**: See `src/.env.example` for all available options
- **Command Line Flags**: All env vars can be overridden with CLI flags
- **Config Priority**: CLI flags > Environment variables > `.env` file

#### Key Configuration Options

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_PORT` | 3000 | HTTP server port for REST mode |
| `APP_DEBUG` | false | Enable debug logging |
| `APP_OS` | Chrome | Device name shown in WhatsApp |
| `APP_BASIC_AUTH` | - | Basic auth credentials (format: `user1:pass1,user2:pass2`) |
| `APP_BASE_PATH` | - | Base path for subpath deployments |
| `DB_URI` | file:storages/whatsapp.db | Main database connection string |
| `DB_KEYS_URI` | file::memory: | Keys database (in-memory by default) |
| `WHATSAPP_AUTO_REPLY` | - | Auto-reply message text |
| `WHATSAPP_AUTO_MARK_READ` | false | Auto-mark incoming messages as read |
| `WHATSAPP_WEBHOOK` | - | Webhook URLs (comma-separated for multiple) |
| `WHATSAPP_WEBHOOK_SECRET` | secret | HMAC secret for webhook verification |
| `WHATSAPP_ACCOUNT_VALIDATION` | true | Validate WhatsApp account exists before sending |
| `WHATSAPP_CHAT_STORAGE` | true | Enable chat history storage |

### Database

- **Main DB**: WhatsApp connection data (SQLite by default, supports PostgreSQL)
- **Chat Storage**: Separate SQLite database for chat history (`storages/chatstorage.db`)
- **Database URIs**: Configurable via `DB_URI` and `DB_KEYS_URI` environment variables

### Mode-Specific Architecture

#### REST Mode
- **Framework**: Fiber web server (v2.52.9)
- **Features**:
  - HTML templates with embedded assets
  - WebSocket support for real-time updates
  - Basic authentication middleware
  - Subpath deployment support
  - Auto-reconnection monitoring
- **Default Port**: 3000 (configurable via `--port` or `APP_PORT`)

#### MCP Mode (Model Context Protocol)
- **Framework**: MCP-Go server (v0.41.1) with SSE transport
- **Features**:
  - AI agent integration via standardized protocol
  - Tool capabilities for WhatsApp operations
  - Resource capabilities for data access
  - Server-Sent Events (SSE) transport
- **Endpoints**:
  - SSE: `http://localhost:8080/sse`
  - Message: `http://localhost:8080/message`
- **Default Port**: 8080 (configurable via `--port` or `--host`)
- **Tool Categories**:
  - **Send Tools**: Send messages, media, contacts, locations
  - **Query Tools**: Chat history, user info, message queries
  - **App Tools**: Login, logout, reconnect, device management
  - **Group Tools**: Create groups, manage participants, settings

### Key Dependencies

- `go.mau.fi/whatsmeow`: WhatsApp Web protocol implementation
- `github.com/gofiber/fiber/v2`: Web framework for REST API
- `github.com/mark3labs/mcp-go`: MCP server implementation
- `github.com/spf13/cobra`: CLI framework
- `github.com/spf13/viper`: Configuration management

### WhatsApp Integration

- **Protocol**: whatsmeow library for WhatsApp Web multi-device protocol
- **Features**:
  - Multi-device WhatsApp account support
  - Auto-reconnection and connection monitoring
  - Media compression (images, videos)
  - Webhook support with HMAC signature verification
  - Account validation before sending
  - Auto-reply and auto-mark-read capabilities
  - Sticker support with automatic WebP conversion
  - Chat history storage (SQLite)
  - Pairing code and QR code login methods
  - Group management and participant operations
  - Newsletter support

## Documentation

### API Documentation
- **Deployment Guide**: `docs/deployment-guide.md` - Complete deployment and usage guide
- **OpenAPI Spec**: `docs/openapi.yaml` - Complete REST API specification (v3.0.0)
- **API Guide for AI Agents**: `docs/openapi.md` - Comprehensive guide for AI agents
- **Webhook Documentation**: `docs/webhook-payload.md` - Webhook payload schemas and integration guide

### Key Documentation Sections
1. **REST API**:
   - All endpoints documented in OpenAPI format
   - Authentication (Basic Auth)
   - Phone number format (JID)
   - Message types and media handling
   - Group operations
   - Chat and message queries

2. **Webhooks**:
   - Event types (messages, receipts, groups, protocol events)
   - HMAC SHA256 signature verification
   - Payload structures for all event types
   - Integration examples (Node.js, Python)
   - Retry logic and error handling

3. **MCP Integration**:
   - Tool capabilities for AI agents
   - SSE transport configuration
   - Available tools and operations

## Recent Features & Updates

### v7.7.1 (Latest)
- **Critical Bug Fix**: Fixed service panic on profile picture fetch
  - Updated whatsmeow library to support PrivacyToken in profile picture requests
  - Prevents service crashes and message loss in downstream systems
  - See: `docs/issues/ISSUE-001-PROFILE-PICTURE-PANIC.md`
- **Documentation**: Added release process guide and CHANGELOG

### v7.7.0
- **Sticker Support**: Automatic conversion of images to WebP format (supports JPG, JPEG, PNG, WebP, GIF)
- **Trusted Proxy**: Support for proxy deployments
- **Document MIME Detection**: Improved document type detection and extension preservation
- **Reconnect Error Handling**: Better session guard and error handling for LoginWithCode

### v7.5.x
- **Group Participants**: List and CSV export functionality
- **Media Download API**: On-demand media download with UI support
- **Group Invite Links**: API endpoint for generating/resetting invite links
- **Memory Optimization**: Improved memory allocation in group operations

### MCP Integration (v7.x+)
- Model Context Protocol server support
- SSE (Server-Sent Events) transport
- Comprehensive tool suite for AI agents
- Resource capabilities for data access

## Important Notes

- The application cannot run both REST and MCP modes simultaneously (limitation from whatsmeow library)
- All source code must be in the `src/` directory
- Media files are stored in `src/statics/media/` and `src/storages/`
- HTML templates and assets are embedded in the binary using Go's embed feature
- FFmpeg is required for media processing (installation varies by platform)
- Version format: `v7.x.x` following [Semantic Versioning](https://semver.org/)
- Release process: See [docs/RELEASE-PROCESS.md](docs/RELEASE-PROCESS.md) for creating new releases
- GitHub Container Registry support available

## Common Troubleshooting

### Media Processing
- Ensure FFmpeg is installed for video/image compression
- Check file size limits: Images (20MB), Files (50MB), Videos (100MB), Downloads (500MB)
- Verify media storage path permissions: `src/statics/media/`

### Database Issues
- Default database: `storages/whatsapp.db` (SQLite)
- Chat storage: `storages/chatstorage.db` (SQLite)
- PostgreSQL supported via `DB_URI` environment variable
- Keys can be stored in-memory (default) or persistent database

### Connection Issues
- Auto-reconnection is enabled by default
- Check WhatsApp session validity with `/app/devices` endpoint
- Use `/app/reconnect` to manually trigger reconnection
- Remote logout clears all session data automatically

### Webhook Issues
- Verify webhook URL is accessible
- Check HMAC signature verification with correct secret
- Review webhook retry logic (5 attempts, exponential backoff)
- Enable debug mode to see detailed webhook logs: `--debug=true`
