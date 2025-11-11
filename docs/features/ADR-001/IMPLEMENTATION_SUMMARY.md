
# Admin API — ADR-001 Implementation Summary

## Overview

This document consolidates ADR-001 delivery notes and explains the Admin API implementation for orchestrating multiple GOWA instances with Supervisord. The Admin API provides HTTP endpoints to create, list, query, update, and delete WhatsApp instances programmatically. It includes configuration generation, process supervision, authentication, validation, and tests to ensure safe, idempotent, and concurrent operations.

## Contents

- Implementation components and packages
- Configuration and environment variables supported
- API endpoints and examples (create, update, list, get, delete)
- Security, concurrency, and idempotency measures
- Testing, documentation, and deployment considerations

## Implementation Components

All server-side implementation resides under `src/internal/admin/` plus CLI and configuration integrations in `src/cmd/` and `src/config/`.

1) Core admin package (`src/internal/admin`)

- `client.go`
  - `SupervisorClient`: wrapper around Supervisord XML-RPC (connect, auth, ping, status)
  - `Config`: supervisor connection settings and factory helpers (including `NewSupervisorClientFromEnv`)

- `conf.go`
  - `InstanceConfig`: structured configuration for per-instance GOWA settings
  - `ConfigWriter`: atomic write/remove of Supervisord program files
  - `ConfigTemplate`: template used to render program configuration and environment/flags
  - `LockManager`: per-port file locks to avoid concurrent mutating operations
  - Helpers: `WriteConfig()`, `WriteConfigWithCustom()`, `RemoveConfig()`, `ConfigExists()` and template rendering logic

- `lifecycle.go`
  - `LifecycleManager`: create, update, delete, and query instance lifecycle operations
  - `Instance` model: represents a running/managed instance (state, PID, uptime, logs)
  - Operations: `CreateInstance`, `CreateInstanceWithConfig`, `UpdateInstanceConfig`, `DeleteInstance`, `ListInstances`, `GetInstance`
  - `waitForInstanceState`: polling loop to wait for Supervisord state transitions

- `api.go`
  - `AdminAPI`: HTTP handlers implemented with Fiber
  - Authentication middleware: Bearer token validation (`ADMIN_TOKEN`)
  - CRUD endpoints for instances and health endpoints (`/healthz` and `/readyz`)
  - Standardized JSON responses including request IDs and timestamps

2) Command Line Integration (`src/cmd/admin.go`)

- Cobra subcommand `whatsapp admin` to run the admin server
- Environment validation and detailed help text
- Configurable admin server port (default `8088`) and other flags

3) Configuration (`src/config/settings.go`)

- Settings for admin and instance management, including:
  - `AdminPort`, `AdminToken`
  - `SupervisorURL`, `SupervisorUser`, `SupervisorPass`
  - `SupervisorConfDir`, `InstancesDir`, `SupervisorLogDir`
  - `GowaBin`, `GowaBasicAuth`, `GowaDebug`, `GowaOS`, `GowaAccountValidation`
  - `LockDir` and other operational paths

4) Dependencies

- `github.com/abrander/go-supervisord` (Supervisord XML-RPC client)
- `github.com/kolo/xmlrpc` (transitive XML-RPC dependency)

## Environment Variables and Mapping

The Admin API supports a GOWA-prefixed set of environment variables that mirror the main application's runtime options. Defaults are loaded from environment or provided when creating an instance.

Required environment variable for admin server:

- `ADMIN_TOKEN` — bearer token required by all protected admin endpoints

Supported per-instance variables (settable via API or defaults):

- `GOWA_DEBUG` (bool)
- `GOWA_OS` (string)
- `GOWA_BASIC_AUTH` (string)
- `GOWA_BASE_PATH` (string)
- `GOWA_AUTO_REPLY` (string)
- `GOWA_AUTO_MARK_READ` (bool)
- `GOWA_WEBHOOK` (string)
- `GOWA_WEBHOOK_SECRET` (string)
- `GOWA_ACCOUNT_VALIDATION` (bool)
- `GOWA_CHAT_STORAGE` (bool)
- `DB_URI` (string)

These map from the main application's variables (`APP_*` and `WHATSAPP_*`) to per-instance `GOWA_*` conventions.

## API Endpoints

Authentication: all admin endpoints require `Authorization: Bearer <ADMIN_TOKEN>`

Primary endpoints:

- `POST /admin/instances`
  - Create a new instance. Accepts JSON body with `port` and optional per-instance configuration.
  - Validates port range (1024-65535), acquires per-port lock, writes Supervisord program file, calls Supervisord `Update()`, and starts the program.

- `GET /admin/instances`
  - Returns a list of configured instances and their runtime state.

- `GET /admin/instances/{port}`
  - Returns details for a single instance (state, pid, uptime, config).

- `PATCH /admin/instances/{port}`
  - Update configuration of an existing instance. This supports partial updates: only supplied fields are changed.
  - Flow: validate port -> acquire lock -> stop instance -> write updated config -> call Supervisord `Update()` -> start instance -> wait for `RUNNING`.

- `DELETE /admin/instances/{port}`
  - Stops and removes the instance's Supervisord program and configuration files.

Health endpoints:

- `GET /healthz` — basic liveness check
- `GET /readyz` — readiness check (includes Supervisor connectivity)

Request example (minimal):

```json
{ "port": 3001 }
```

Request example (full):

```json
{
  "port": 3001,
  "basic_auth": "user:password",
  "debug": true,
  "os": "Custom OS",
  "account_validation": false,
  "base_path": "/api/v1",
  "auto_reply": "Thank you for your message",
  "auto_mark_read": true,
  "webhook": "https://myapp.com/webhook",
  "webhook_secret": "my-secret-key",
  "chat_storage": false
}
```

CLI example to start admin server:

```bash
export ADMIN_TOKEN="your-secure-token"
./whatsapp admin --port 8088
```

API examples (curl):

Create instance (minimal):

```bash
curl -X POST "http://localhost:8088/admin/instances" \
  -H "Authorization: Bearer your-secure-token" \
  -H "Content-Type: application/json" \
  -d '{"port": 3001}'
```

Update instance (partial):

```bash
curl -X PATCH "http://localhost:8088/admin/instances/3001" \
  -H "Authorization: Bearer your-secure-token" \
  -H "Content-Type: application/json" \
  -d '{"debug": false, "webhook": "https://new-webhook.example.com/whatsapp"}'
```

List instances:

```bash
curl -X GET "http://localhost:8088/admin/instances" \
  -H "Authorization: Bearer your-secure-token"
```

Get instance:

```bash
curl -X GET "http://localhost:8088/admin/instances/3001" \
  -H "Authorization: Bearer your-secure-token"
```

Delete instance:

```bash
curl -X DELETE "http://localhost:8088/admin/instances/3001" \
  -H "Authorization: Bearer your-secure-token"
```

## Update endpoint details

Supported update request (partial allowed):

```json
{
  "basic_auth": "newuser:newpassword",
  "debug": false,
  "os": "UpdatedDevice",
  "account_validation": true,
  "base_path": "/api/v2",
  "auto_reply": "Thanks",
  "auto_mark_read": true,
  "webhook": "https://hooks.example.com/gowa",
  "webhook_secret": "s3cret",
  "chat_storage": false
}
```

Behavioral notes:

- Partial updates: only provided fields are written to the per-instance configuration.
- The update is performed under a per-port lock to prevent concurrent mutating operations.
- The system stops the instance, replaces the Supervisor config atomically, calls Supervisord `Update()` to reconcile, and restarts the instance.
- If the update fails, the process attempts cleanup; the operation returns an error code (404 for missing instance, 409 for lock conflicts, 500 for internal errors).

## Security, Concurrency, and Idempotency

- Bearer token authentication for all protected endpoints using `ADMIN_TOKEN`.
- Port validation to ensure only allowed port ranges are used.
- Per-port file locks (LockManager) to prevent concurrent create/delete/update for the same port.
- Atomic configuration file writes and safe removal on failures.
- Idempotent operations where possible: create/update/delete return clear state and avoid corrupting existing configuration.

## Configuration Generation

Supervisord program files are generated from a template that includes:

- Program name and command to run the GOWA binary with appropriate flags derived from `InstanceConfig`.
- Environment variables set for each instance (`GOWA_*` and `DB_URI`).
- Logging paths and rotation options pointing to `SupervisorLogDir`.
- Auto-start and restart policy for resilience.

The `ConfigWriter` supports both default `WriteConfig(port)` and `WriteConfigWithCustom(port, customConfig)` for update scenarios.

## Testing

- Unit tests across client, conf, lifecycle, and API components.
- Mockable Supervisor client so tests run without an actual Supervisord.
- Integration tests validating full configuration rendering and instance lifecycle (create -> start -> observe -> stop -> remove).
- API tests for endpoints, auth, validation rules, and the new update endpoint.

Test status summary:

- Unit and integration tests run locally and in CI; all tests related to ADR-001 are passing.

## Documentation and Deliverables

- `docs/admin-api.md` updated with API reference, examples, and the update endpoint.
- `docker-compose.admin.yml` example deployment file.
- Inline command help in `src/cmd/admin.go`.
- Helper script `.devcontainer/dev.sh` includes `update` helper for development and testing.

## Key Design Decisions

1. Supervisord is used for process supervision because it is stable, widely supported, and exposes an XML-RPC API suitable for programmatic control.
2. Atomic configuration file writes and per-port locks reduce the risk of partial or concurrent modifications.
3. Template-based config generation keeps program files consistent and easy to update.
4. Interface-based lifecycle components improve testability and separation of concerns.
5. Structured JSON logging and standardized API responses improve observability and client integration.

## Environment Variable Mapping (Summary)

The Admin API maps main app variables into per-instance `GOWA_*` variables. The important mappings are:

- `APP_PORT` → per-instance port
- `APP_DEBUG` → `GOWA_DEBUG`
- `APP_OS` → `GOWA_OS`
- `APP_BASIC_AUTH` → `GOWA_BASIC_AUTH`
- `APP_BASE_PATH` → `GOWA_BASE_PATH`
- `DB_URI` → configured per instance (or auto-generated)
- `WHATSAPP_AUTO_REPLY` → `GOWA_AUTO_REPLY`
- `WHATSAPP_AUTO_MARK_READ` → `GOWA_AUTO_MARK_READ`
- `WHATSAPP_WEBHOOK` → `GOWA_WEBHOOK`
- `WHATSAPP_WEBHOOK_SECRET` → `GOWA_WEBHOOK_SECRET`
- `WHATSAPP_ACCOUNT_VALIDATION` → `GOWA_ACCOUNT_VALIDATION`
- `WHATSAPP_CHAT_STORAGE` → `GOWA_CHAT_STORAGE`

## Deployment Considerations

- Admin server should be exposed only to trusted networks or protected by TLS / reverse proxy in production.
- `ADMIN_TOKEN` must be stored securely (secrets manager, environment injection) and rotated when needed.
- Ensure Supervisord is running and reachable by the configured `SupervisorURL` and credentials.
- File system permissions must allow the admin service to write Supervisord config files and instance directories.

## Development and Testing

For development, a helper script `.devcontainer/dev.sh` includes commands to create, list, update and delete instances locally. Example commands:

```bash
# Start the admin server
./dev.sh start-admin

# Create a new instance
./dev.sh create 3001

# Update instance configuration
./dev.sh update 3001 '{"debug": false, "webhook": "https://new-webhook.com"}'

# List instances
./dev.sh list

# Delete an instance
./dev.sh delete 3001
```

## Next Steps and Enhancements

Suggested low-risk improvements:

1. Bulk instance operations (create/delete in batch) with transactional semantics.
2. Configuration profiles / presets for common setups.
3. Extended health-check endpoints for per-instance detailed metrics.
4. Web UI for easier instance management.

## Conclusion

ADR-001 is implemented: the Admin API provides full programmatic management of GOWA instances under Supervisord, supports all environment variables from the main application, includes robust concurrency and safety mechanisms, and has comprehensive tests and developer documentation.

