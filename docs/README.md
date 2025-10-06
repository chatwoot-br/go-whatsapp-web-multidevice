# API Documentation Structure

This directory contains the OpenAPI specifications for the WhatsApp API MultiDevice project.

## Files

### `openapi.yaml`
The main OpenAPI specification for the WhatsApp API, containing:
- Authentication endpoints (`/app/login`, `/app/logout`, etc.)
- User management (`/user/info`, `/user/avatar`, etc.)
- Message sending (`/send/message`, `/send/image`, etc.)
- Message manipulation (`/message/{id}/revoke`, `/message/{id}/reaction`, etc.)
- Chat management (`/chats`, `/chat/{id}/messages`, etc.)
- Group management (`/group/create`, `/group/participants`, etc.)
- Newsletter management (`/newsletter/unfollow`, etc.)

**Authentication**: Basic Auth  
**Default Server**: `http://localhost:3000`

### `admin-api-openapi.yaml`
Dedicated OpenAPI specification for the Admin API, containing:
- Instance management (`/admin/instances` - POST, GET)
- Individual instance operations (`/admin/instances/{port}` - GET, PATCH, DELETE)
- Health checks (`/healthz`, `/readyz`)

**Authentication**: Bearer Token (requires `ADMIN_TOKEN`)  
**Default Server**: `http://localhost:8088`

## Why Separate Files?

1. **Clarity**: Each API serves different purposes and audiences
2. **Authentication**: Different authentication methods (Basic Auth vs Bearer Token)
3. **Deployment**: APIs may be deployed on different servers/ports
4. **Maintenance**: Easier to maintain and update each specification independently
5. **Documentation**: Cleaner documentation for each specific use case

## Usage

### Main WhatsApp API
```bash
# View the specification
open docs/openapi.yaml

# Or serve with a tool like swagger-ui
npx @apidevtools/swagger-cli serve docs/openapi.yaml
```

### Admin API
```bash
# View the specification
open docs/admin-api-openapi.yaml

# Or serve with a tool like swagger-ui
npx @apidevtools/swagger-cli serve docs/admin-api-openapi.yaml
```

## Integration

When using API documentation tools or generating SDKs, you can:

1. Use each specification independently for focused documentation
2. Combine both specifications if you need complete API coverage
3. Generate separate client SDKs for different use cases

## Related Documentation

- [Admin API Implementation Details](features/ADR-001/IMPLEMENTATION_SUMMARY.md)
- [Admin API Usage Guide](admin-api.md)
- [Main API Usage Examples](../gallery/)
