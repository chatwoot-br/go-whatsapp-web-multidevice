# Contributing to go-whatsapp-web-multidevice

Thank you for your interest in contributing! Please see our comprehensive [Contributing Guide](docs/developer/contributing.md) for detailed information on:

- Development setup
- Coding standards
- Testing requirements
- Commit message format
- Pull request process

## Quick Links

- [Full Contributing Guide](docs/developer/contributing.md)
- [Architecture Overview](docs/developer/architecture.md)
- [Development Commands](CLAUDE.md#common-development-commands)
- [API Documentation](docs/reference/api/)

## Quick Start

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/go-whatsapp-web-multidevice.git

# 2. Install dependencies
cd src && go mod download

# 3. Run tests
go test ./...

# 4. Build and run
go build -o whatsapp
./whatsapp rest
```

## Commit Format

Use conventional commits:

```
<type>(<scope>): <description>
```

Examples:
- `feat(send): add message scheduling`
- `fix(webhook): correct HMAC validation`
- `docs(api): update examples`

## Need Help?

- Questions: [GitHub Discussions](https://github.com/aldinokemal/go-whatsapp-web-multidevice/discussions)
- Bugs: [GitHub Issues](https://github.com/aldinokemal/go-whatsapp-web-multidevice/issues)

We appreciate your contributions!
