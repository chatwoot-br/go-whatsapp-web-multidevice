# Architecture Overview

This document describes the architecture and design patterns used in the go-whatsapp-web-multidevice project.

## Table of Contents

- [High-Level Architecture](#high-level-architecture)
- [Design Patterns](#design-patterns)
- [Layer Descriptions](#layer-descriptions)
- [Data Flow](#data-flow)
- [Key Components](#key-components)
- [Technology Stack](#technology-stack)

## High-Level Architecture

The application follows **Clean Architecture** principles with **Domain-Driven Design (DDD)** patterns, ensuring clear separation of concerns and maintainability.

```
┌─────────────────────────────────────────────────────────────┐
│                        Entry Point                           │
│                       (main.go + cmd/)                       │
│                      Cobra CLI Framework                     │
└──────────────────────┬───────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
    ┌────────┐   ┌────────┐   ┌────────┐
    │  REST  │   │  MCP   │   │ Admin  │
    │  Mode  │   │  Mode  │   │  Mode  │
    └────┬───┘   └────┬───┘   └────┬───┘
         │            │            │
         └────────────┼────────────┘
                      │
         ┌────────────▼────────────┐
         │      UI Layer           │
         │  (Fiber/MCP/WebSocket)  │
         └────────────┬────────────┘
                      │
         ┌────────────▼────────────┐
         │    Usecase Layer        │
         │  (Business Logic)       │
         └────────────┬────────────┘
                      │
         ┌────────────▼────────────┐
         │    Domain Layer         │
         │  (Business Entities)    │
         └────────────┬────────────┘
                      │
         ┌────────────▼────────────┐
         │  Infrastructure Layer   │
         │  (WhatsApp/Database)    │
         └─────────────────────────┘
```

## Design Patterns

### 1. Clean Architecture

The application is organized into distinct layers with clear dependencies:

- **Outer layers depend on inner layers** (Dependency Rule)
- **Inner layers have no knowledge of outer layers**
- **Business logic is independent of frameworks and UI**

### 2. Domain-Driven Design

Business logic is organized into **domain packages** representing bounded contexts:

- `app` - Application lifecycle (login, logout, device management)
- `chat` - Chat operations and queries
- `group` - Group management and participants
- `message` - Message operations and history
- `send` - Sending messages and media
- `user` - User information and profile
- `newsletter` - Newsletter/channel operations
- `chatstorage` - Chat history storage

### 3. Dependency Injection

- **Interfaces define contracts** between layers
- **Concrete implementations injected at runtime**
- Enables testing and loose coupling

Example:
```go
// Interface definition in domain
type IAppUsecase interface {
    Login() error
    Logout() error
    Reconnect() error
}

// Concrete implementation in usecase
type appUsecase struct {
    service domainApp.IAppService
}

// Injection in cmd/root.go
appUsecase = usecase.NewAppUsecase(appService)
```

### 4. Repository Pattern

Data access abstracted through repository interfaces:

```go
type IChatStorageRepository interface {
    GetMessages(ctx context.Context, jid string) ([]Message, error)
    SaveMessage(ctx context.Context, msg Message) error
}
```

### 5. Command Pattern

CLI commands encapsulate different modes of operation:

- `rest` - HTTP REST API server
- `mcp` - Model Context Protocol server
- `admin` - Multi-instance admin API

## Layer Descriptions

### Entry Point Layer (`main.go`, `cmd/`)

**Purpose**: Application initialization and command routing

**Key Files**:
- `main.go` - Entry point, embeds static assets
- `cmd/root.go` - Root command, dependency injection
- `cmd/rest.go` - REST API server command
- `cmd/mcp.go` - MCP server command
- `cmd/admin.go` - Admin API command

**Responsibilities**:
- Parse CLI flags and environment variables
- Initialize all dependencies (database, WhatsApp client, services)
- Route to appropriate command (rest/mcp/admin)
- Manage application lifecycle

**Key Dependencies**:
- Cobra (CLI framework)
- Viper (configuration management)

### UI Layer (`ui/`)

**Purpose**: Handle user interactions and external interfaces

**Packages**:

1. **`ui/rest/`** - HTTP REST API
   - Fiber web framework
   - Routes and handlers
   - Middleware (auth, logging, CORS)
   - Request/response serialization

2. **`ui/mcp/`** - Model Context Protocol Server
   - SSE (Server-Sent Events) transport
   - Tool registration and handlers
   - Resource capabilities
   - AI agent integration

3. **`ui/websocket/`** - WebSocket Server
   - Real-time event streaming
   - Connection management
   - Event broadcasting

**Responsibilities**:
- Request validation
- Authentication/authorization
- Response formatting
- Error handling
- Static file serving (REST mode)

### Usecase Layer (`usecase/`)

**Purpose**: Orchestrate business logic and coordinate between domains

**Key Files**:
- `app.go` - Application operations
- `chat.go` - Chat operations
- `group.go` - Group management
- `message.go` - Message handling
- `send.go` - Send operations
- `user.go` - User operations
- `newsletter.go` - Newsletter operations

**Responsibilities**:
- Coordinate multiple domain services
- Implement complex workflows
- Transaction management
- Business rules enforcement
- Data transformation between layers

**Example Flow**:
```go
// Usecase coordinates multiple domain services
func (u *sendUsecase) SendTextMessage(ctx context.Context, req SendTextRequest) error {
    // 1. Validate user exists (user domain)
    if err := u.userService.ValidatePhone(req.Phone); err != nil {
        return err
    }

    // 2. Send message (send domain)
    if err := u.sendService.SendText(ctx, req); err != nil {
        return err
    }

    // 3. Store in history (chatstorage domain)
    return u.chatStorageService.SaveMessage(ctx, msg)
}
```

### Domain Layer (`domains/`)

**Purpose**: Core business logic and entities

**Structure per Domain**:
```
domains/
├── app/
│   ├── service.go       # Domain service interface
│   ├── service_impl.go  # Domain service implementation
│   └── entity.go        # Domain entities/models
├── chat/
├── group/
├── message/
├── send/
└── user/
```

**Responsibilities**:
- Define business entities (structs)
- Define service interfaces
- Implement core business rules
- Domain-specific validations
- No dependencies on outer layers

**Example**:
```go
// Domain entity
type Message struct {
    ID        string
    From      string
    To        string
    Content   string
    Timestamp time.Time
}

// Domain service interface
type IMessageService interface {
    GetMessageByID(ctx context.Context, id string) (*Message, error)
    SearchMessages(ctx context.Context, criteria SearchCriteria) ([]Message, error)
}
```

### Infrastructure Layer (`infrastructure/`)

**Purpose**: External integrations and data persistence

**Packages**:

1. **`infrastructure/whatsapp/`** - WhatsApp Web Protocol
   - whatsmeow client wrapper
   - Event handlers
   - Connection management
   - Media processing

2. **`infrastructure/chatstorage/`** - Chat History Storage
   - SQLite repository implementation
   - Message persistence
   - Query optimization

**Responsibilities**:
- Implement repository interfaces
- Handle external API calls
- Manage connections and sessions
- Data persistence
- Media file operations

### Shared Utilities (`pkg/`)

**Purpose**: Reusable utilities and helpers

**Packages**:
- `utils/` - Common utilities
- `whatsapp/` - WhatsApp-specific helpers
- Configuration loaders
- Validation helpers

## Data Flow

### Request Flow (REST API)

```
1. HTTP Request
   │
   ▼
2. Fiber Router (ui/rest/)
   │ - Route matching
   │ - Middleware execution (auth, logging)
   ▼
3. Handler (ui/rest/)
   │ - Request validation
   │ - DTO parsing
   ▼
4. Usecase (usecase/)
   │ - Business logic orchestration
   │ - Coordinate multiple domains
   ▼
5. Domain Service (domains/)
   │ - Core business rules
   │ - Entity operations
   ▼
6. Infrastructure (infrastructure/)
   │ - WhatsApp API calls
   │ - Database operations
   ▼
7. Response
   │ - Domain entity
   ▼
8. Usecase
   │ - Data transformation
   ▼
9. Handler
   │ - Response formatting
   │ - Error handling
   ▼
10. HTTP Response
```

### Event Flow (WhatsApp Events)

```
1. WhatsApp Event
   │
   ▼
2. whatsmeow Event Handler (infrastructure/whatsapp/)
   │ - Event parsing
   │ - Event type detection
   ▼
3. Domain Service (domains/)
   │ - Process event
   │ - Update state
   ▼
4. Storage (infrastructure/chatstorage/)
   │ - Persist message/event
   ▼
5. Webhook Dispatcher (infrastructure/whatsapp/)
   │ - Format payload
   │ - HMAC signature
   │ - HTTP POST to webhook URLs
   ▼
6. WebSocket Broadcast (ui/websocket/)
   │ - Send to connected clients
```

## Key Components

### 1. WhatsApp Client

**Location**: `infrastructure/whatsapp/`

**Responsibilities**:
- Maintain WhatsApp connection
- Handle multi-device protocol
- Process incoming events
- Send messages and media
- Manage pairing/QR code authentication

**Key Features**:
- Auto-reconnection
- Event handlers registration
- Media download/upload
- Group operations
- Profile management

### 2. Configuration Management

**Location**: `config/`, `cmd/root.go`

**Approach**:
- Environment variables (via `.env` file)
- CLI flags (via Cobra)
- Priority: CLI flags > Environment > Defaults

**Key Settings**:
- `APP_PORT` - Server port
- `APP_DEBUG` - Debug logging
- `DB_URI` - Database connection
- `WHATSAPP_WEBHOOK` - Webhook URLs
- `WHATSAPP_AUTO_REPLY` - Auto-reply message

### 3. Database

**Main Database** (WhatsApp session):
- Default: SQLite (`storages/whatsapp.db`)
- Supported: PostgreSQL
- Stores: Device info, session keys, contacts

**Chat Storage** (Message history):
- SQLite (`storages/chatstorage.db`)
- Optional feature (configurable)
- Stores: Messages, media metadata

### 4. Media Handling

**Location**: `domains/send/`, `infrastructure/whatsapp/`

**Flow**:
1. Receive media URL or base64
2. Download/decode media
3. Validate file type and size
4. Compress (if needed, using FFmpeg)
5. Upload to WhatsApp servers
6. Send message with media ID

**Supported Formats**:
- Images: JPG, PNG, WebP (auto-converted to WebP for stickers)
- Videos: MP4, AVI, MKV (compressed)
- Audio: MP3, OGG, AAC, WAV, OPUS (auto-converted)
- Documents: All formats

### 5. Webhook System

**Location**: `infrastructure/whatsapp/`

**Features**:
- Multiple webhook URLs support
- HMAC SHA256 signature verification
- Retry logic (5 attempts, exponential backoff)
- Event filtering

**Event Types**:
- `message` - Incoming messages
- `receipt` - Message receipts (sent, delivered, read)
- `group` - Group events (join, leave, update)
- `call` - Call events

## Technology Stack

### Core

- **Language**: Go 1.21+
- **WhatsApp Protocol**: whatsmeow
- **CLI Framework**: Cobra + Viper
- **Web Framework**: Fiber v2
- **MCP Server**: mcp-go
- **Database**: SQLite / PostgreSQL

### External Tools

- **FFmpeg** - Media processing (compression, conversion)
- **ImageMagick** (optional) - Image manipulation

### Key Libraries

- `go.mau.fi/whatsmeow` - WhatsApp Web multi-device protocol
- `github.com/gofiber/fiber/v2` - HTTP web framework
- `github.com/mark3labs/mcp-go` - Model Context Protocol
- `github.com/spf13/cobra` - CLI commands
- `github.com/spf13/viper` - Configuration management
- `github.com/mattn/go-sqlite3` - SQLite driver
- `github.com/lib/pq` - PostgreSQL driver

## Architectural Decisions

For detailed architecture decisions, see the [ADR directory](adr/).

Key decisions documented:
- [ADR-0001: Admin API Architecture](adr/0001-admin-api.md) - Multi-instance management with supervisord

## Design Principles

1. **Separation of Concerns**: Each layer has a clear responsibility
2. **Dependency Inversion**: Depend on interfaces, not concrete implementations
3. **Single Responsibility**: Each package/module has one reason to change
4. **Interface Segregation**: Small, focused interfaces
5. **DRY (Don't Repeat Yourself)**: Shared utilities in `pkg/`
6. **Testability**: Dependency injection enables unit testing
7. **Fail Fast**: Early validation and error handling
8. **Idiomatic Go**: Follow Go best practices and conventions

## Testing Strategy

- **Unit Tests**: Test domain logic in isolation
- **Integration Tests**: Test database and WhatsApp interactions
- **E2E Tests**: Test complete workflows via API

See [Testing Guide](testing.md) for details.

## Performance Considerations

1. **Connection Pooling**: Database connection pools
2. **Concurrent Processing**: Goroutines for webhook dispatching
3. **Caching**: In-memory caching for frequent queries
4. **Media Compression**: FFmpeg for reducing file sizes
5. **Efficient Queries**: Indexed database queries

See [Performance Tuning Guide](../operations/performance-tuning.md) for optimization strategies.

## Security

1. **Authentication**: Basic Auth for API access
2. **Webhook Security**: HMAC signature verification
3. **Input Validation**: Strict validation in UI layer
4. **No Sensitive Data Logging**: Debug mode excludes secrets
5. **Session Security**: Encrypted WhatsApp session storage

See [Security Best Practices](../operations/security-best-practices.md) for detailed guidelines.

## Deployment Modes

### 1. REST API Mode

```bash
./whatsapp rest --port 3000
```

- HTTP server with web UI
- REST API endpoints
- WebSocket for real-time updates
- Basic auth support

### 2. MCP Server Mode

```bash
./whatsapp mcp --port 8080
```

- Model Context Protocol server
- SSE transport
- AI agent integration
- No web UI

### 3. Admin Mode

```bash
./whatsapp admin --port 9000
```

- Multi-instance management
- Process lifecycle control
- Supervisord integration

**Note**: Only one mode can run at a time due to WhatsApp client limitations.

## Future Enhancements

Potential architectural improvements:

1. **Message Queue**: Add Redis/RabbitMQ for webhook delivery
2. **Distributed Tracing**: OpenTelemetry integration
3. **Metrics**: Prometheus metrics export
4. **Rate Limiting**: Protect against abuse
5. **Load Balancing**: Multiple instance support (with shared session storage)
6. **Event Sourcing**: Full event history with replay capability

## Related Documentation

- [Contributing Guide](contributing.md) - How to contribute
- [Release Process](release-process.md) - Creating releases
- [API Reference](../reference/api/) - API specifications
- [Operations Guide](../operations/) - Deployment and monitoring
