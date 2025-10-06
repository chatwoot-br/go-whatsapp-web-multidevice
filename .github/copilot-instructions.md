---
description: complete guide about project structure
globs: 
alwaysApply: false
---
# Go WhatsApp Web Multidevice API

This is a Go implementation of a WhatsApp Web Multidevice API that allows you to interact with WhatsApp through HTTP APIs.

## Project Structure

### Root Directory

- [readme.md](mdc:readme.md) - Project documentation and usage instructions
- [docker-compose.yml](mdc:docker-compose.yml) - Docker configuration for running the application
- [LICENCE.txt](mdc:LICENCE.txt) - License information

### Source Code (`src/`)

The main source code is organized in the `src` directory with the following structure:

#### Command Line Interface

- [src/cmd/root.go](mdc:src/cmd/root.go) - Root command using Cobra for CLI commands, handles configuration loading
- [src/cmd/rest.go](mdc:src/cmd/rest.go) - REST server command implementation
- [src/cmd/mcp.go](mdc:src/cmd/mcp.go) - Model Context Protocol server command implementation

#### Configuration

- [src/config/](mdc:src/config) - Application configuration settings and constants

#### Domain Models

The application is organized using domain-driven design principles:

- [src/domains/app/](mdc:src/domains/app) - Core application domain models
- [src/domains/chat/](mdc:src/domains/chat) - Chat-related domain models and interfaces
- [src/domains/group/](mdc:src/domains/group) - Group-related domain models
- [src/domains/message/](mdc:src/domains/message) - Message-related domain models
- [src/domains/newsletter/](mdc:src/domains/newsletter) - Newsletter-related domain models
- [src/domains/send/](mdc:src/domains/send) - Message sending domain models
- [src/domains/user/](mdc:src/domains/user) - User-related domain models

#### Infrastructure

- [src/infrastructure/chatstorage/](mdc:src/infrastructure/chatstorage) - Chat storage repository and WhatsApp integration
- [src/infrastructure/whatsapp/](mdc:src/infrastructure/whatsapp) - WhatsApp client implementation and related infrastructure

#### User Interface

- [src/ui/rest/](mdc:src/ui/rest) - REST API implementation
  - [src/ui/rest/helpers/](mdc:src/ui/rest/helpers) - Helper functions for REST handlers
  - [src/ui/rest/middleware/](mdc:src/ui/rest/middleware) - Middleware components for request processing
- [src/ui/websocket/](mdc:src/ui/websocket) - WebSocket implementation for real-time communication
- [src/ui/mcp/](mdc:src/ui/mcp) - Model Context Protocol server to communication with AI Agent

#### Utilities and Shared Components

- [src/pkg/error/](mdc:src/pkg/error) - Error handling utilities
- [src/pkg/utils/](mdc:src/pkg/utils) - General utility functions

#### Use Cases

- [src/usecase/](mdc:src/usecase) - Application services that implement business logic

#### Static Resources

- [src/statics/](mdc:src/statics) - Static resources like media files
  - [src/statics/media/](mdc:src/statics/media) - Media files
  - [src/statics/qrcode/](mdc:src/statics/qrcode) - QR code images for WhatsApp authentication
  - [src/statics/senditems/](mdc:src/statics/senditems) - Items to be sent via WhatsApp

#### Validation

- [src/validations/](mdc:src/validations) - Request validation logic

#### Views

- [src/views/](mdc:src/views) - Templates and UI components
  - [src/views/assets/](mdc:src/views/assets) - Frontend assets (CSS, JS, etc.)
  - [src/views/components/](mdc:src/views/components) - Reusable UI components
    - [src/views/components/generic/](mdc:src/views/components/generic) - Generic UI components

## Admin API (Supervisor-based instance management)

Add a concise summary of the Admin API used to manage multiple GOWA instances under Supervisord. See `docs/features/ADR-001/` and `docs/admin-api.md` for full design and examples; this section highlights the most important points for contributors and automation.

- Purpose: expose an HTTP Admin API to create, list, update, and delete GOWA instances managed by Supervisord.
- CLI: `whatsapp admin` subcommand starts the admin server (default port `8088`).
- Primary endpoints: `POST /admin/instances`, `GET /admin/instances`, `GET /admin/instances/{port}`, `PATCH /admin/instances/{port}`, `DELETE /admin/instances/{port}`. Health endpoints: `/healthz`, `/readyz`.
- Authentication: protected endpoints require `Authorization: Bearer <ADMIN_TOKEN>`. The admin server should refuse to start in production without `ADMIN_TOKEN` configured.
- Supervisor integration: write per-instance supervisor program files to `SUPERVISOR_CONF_DIR` (default `/etc/supervisor/conf.d`), and control processes via Supervisord XML-RPC (`Update()`, `StartProcess`, `StopProcess`, `RemoveProcessGroup`).
- Instance layout: per-instance data at `INSTANCES_DIR` (default `/app/instances/<PORT>/storages`) and logs under `SUPERVISOR_LOG_DIR` (default `/var/log/supervisor`).
- Safety practices: atomic config writes (write tmp -> fsync -> rename), per-port locks to prevent concurrent mutations, idempotent flows, and port validation (allow only 1024-65535, ensure not already claimed).
- Supervisor RPC binding: prefer UNIX socket or loopback (`127.0.0.1:9001/RPC2`); never expose Supervisord RPC on public interfaces.
- Documentation: detailed docs and examples live in `docs/features/ADR-001/IMPLEMENTATION_SUMMARY.md` and `docs/admin-api.md`.

Use this summary when generating code, docs, or tests that interact with the Admin API.

### Documentation

- [docs/](mdc:docs) - Project documentation
  - [docs/openapi.yaml](mdc:docs/openapi.yaml) - OpenAPI specification for the REST API
  - [docs/sdk/](mdc:docs/sdk) - SDK documentation

## Documentation language rule

All written content added to this repository (source code comments, documentation files, READMEs, ADRs, and generated instructions) MUST be written in English. This helps keep documentation consistent, searchable, and accessible to the project's international audience and tooling. If a non-English example is required for a test case, include a clear inline comment explaining the reason and keep the primary documentation text in English.

### Docker

- [docker/](mdc:docker) - Docker-related files and configurations
  - [docker/golang.Dockerfile](mdc:docker/golang.Dockerfile) - Dockerfile for building the Go application

## Key Application Features

- WhatsApp login via QR code or pairing code
- Send/receive messages, media, contacts, locations
- Group management features
- Newsletter management
- WebSocket real-time updates
- Webhooks for message events
- Auto-reply functionality
- Model Context Protocol (MCP) server for AI agent communication

## Application Flow

### REST Server Mode

1. The application starts from [src/cmd/root.go](mdc:src/cmd/root.go)
2. REST server command is executed via [src/cmd/rest.go](mdc:src/cmd/rest.go)
3. Configuration is loaded from environment variables or command line flags
4. The REST server is initialized using Fiber framework
5. WhatsApp client is initialized and services are created
6. REST routes are registered for different domains
7. WebSocket hub is started for real-time communication
8. Background tasks are started (auto-reconnect, chat storage flushing)
9. The server listens for requests on the configured port

### MCP Server Mode

1. The application starts from [src/cmd/root.go](mdc:src/cmd/root.go)
2. MCP server command is executed via [src/cmd/mcp.go](mdc:src/cmd/mcp.go)
3. Model Context Protocol server is initialized for AI agent communication
4. WhatsApp client services are made available through MCP protocol
5. The MCP server listens for AI agent requests
