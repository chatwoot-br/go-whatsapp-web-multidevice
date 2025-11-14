# Developer Documentation

Resources for contributors and developers working on this project.

## Getting Started

- **[Architecture Overview](architecture.md)** - System architecture and design patterns
- **[Contributing Guide](contributing.md)** - How to contribute to the project
- **[Testing Guide](testing.md)** - Running and writing tests

## Release Management

- **[Release Process](release-process.md)** - Creating new releases

## Architecture Decision Records

- **[ADR-0001: Admin API Architecture](adr/0001-admin-api.md)** - Multi-instance management design

## Development Workflow

### Building and Running

```bash
# Build
cd src && go build -o whatsapp

# Run REST mode
cd src && go run . rest

# Run MCP mode
cd src && go run . mcp

# Run tests
cd src && go test ./...
```

### Code Standards

- Follow Go standard practices (gofmt, golint)
- Write tests for new features
- Update documentation for API changes
- Follow conventional commit format

## Project Structure

```
src/
├── cmd/           # CLI commands
├── domains/       # Business domain logic
├── infrastructure/ # External integrations
├── ui/            # User interface layers
├── usecase/       # Application use cases
└── validations/   # Input validation
```

## Related Documentation

- **[API Reference](../reference/)** - Complete API specifications
- **[Operations Guide](../operations/)** - Deployment and monitoring
