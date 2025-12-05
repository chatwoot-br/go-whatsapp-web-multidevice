# Contributing Guide

Thank you for considering contributing to go-whatsapp-web-multidevice! This guide will help you get started.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)
- [Documentation](#documentation)

## Code of Conduct

By participating in this project, you agree to:

- Be respectful and inclusive
- Accept constructive criticism gracefully
- Focus on what is best for the community
- Show empathy towards other community members

## Getting Started

### Prerequisites

- **Go 1.21 or later**
- **Git**
- **FFmpeg** (for media processing)
- **Docker** (optional, for testing)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:

```bash
git clone https://github.com/YOUR_USERNAME/go-whatsapp-web-multidevice.git
cd go-whatsapp-web-multidevice
```

3. Add upstream remote:

```bash
git add remote upstream https://github.com/aldinokemal/go-whatsapp-web-multidevice.git
```

### Development Setup

1. Install dependencies:

```bash
cd src
go mod download
```

2. Copy environment file:

```bash
cp .env.example .env
```

3. Build the project:

```bash
cd src
go build -o whatsapp
```

4. Run tests:

```bash
cd src
go test ./...
```

5. Run the application:

```bash
cd src
go run . rest
```

The API will be available at `http://localhost:3000`.

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b username/feature-name
```

**Branch Naming Convention**:
- `username/feature-name` - New features
- `username/fix-issue-123` - Bug fixes
- `username/docs-update` - Documentation updates

Examples:
- `john/add-message-scheduling`
- `jane/fix-media-upload`
- `bob/docs-api-reference`

### 2. Make Changes

- Write clean, well-documented code
- Follow Go best practices
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run all tests
cd src && go test ./...

# Run tests with coverage
cd src && go test -cover ./...

# Run specific package tests
cd src && go test ./validations

# Format code
cd src && go fmt ./...

# Check for issues
cd src && go vet ./...
```

### 4. Commit Your Changes

```bash
git add .
git commit -m "feat(send): add message scheduling feature"
```

See [Commit Messages](#commit-messages) for format details.

### 5. Push and Create Pull Request

```bash
git push origin username/feature-name
```

Then create a Pull Request on GitHub.

## Coding Standards

### Go Style Guide

Follow standard Go conventions:

- **gofmt**: All code must be formatted with `go fmt`
- **golint**: Code should pass `golint` checks
- **go vet**: Code should pass `go vet` checks

### Project-Specific Standards

1. **Package Organization**

Follow the existing architecture:
```
src/
├── cmd/           # CLI commands
├── domains/       # Business domain logic
├── infrastructure/ # External integrations
├── ui/            # User interface layers
├── usecase/       # Application use cases
└── pkg/           # Shared utilities
```

2. **Naming Conventions**

- **Interfaces**: Prefix with `I` (e.g., `IUserService`)
- **Structs**: PascalCase (e.g., `UserService`)
- **Functions**: camelCase for private, PascalCase for exported
- **Constants**: PascalCase or UPPER_SNAKE_CASE

3. **Error Handling**

```go
// Bad
if err != nil {
    log.Println(err)
}

// Good
if err != nil {
    return fmt.Errorf("failed to send message: %w", err)
}
```

4. **Context Usage**

Always pass `context.Context` as the first parameter:

```go
func SendMessage(ctx context.Context, phone string, message string) error {
    // Implementation
}
```

5. **Logging**

Use structured logging with logrus:

```go
log.WithFields(log.Fields{
    "phone": phone,
    "type":  "text",
}).Info("Message sent successfully")
```

6. **Comments**

- Add package comments for all packages
- Comment exported functions, types, and constants
- Use `//` for single-line comments
- Use `/* */` for multi-line comments

```go
// SendTextMessage sends a text message to a WhatsApp user or group.
// It validates the phone number, checks if the account exists (if validation is enabled),
// and sends the message through the WhatsApp client.
func (s *sendService) SendTextMessage(ctx context.Context, req SendTextRequest) error {
    // Implementation
}
```

## Testing

### Writing Tests

1. **Test File Naming**: `*_test.go`
2. **Test Function Naming**: `TestFunctionName`
3. **Test Organization**: Table-driven tests preferred

Example:

```go
func TestSendTextMessage(t *testing.T) {
    tests := []struct {
        name    string
        phone   string
        message string
        wantErr bool
    }{
        {
            name:    "valid message",
            phone:   "6281234567890",
            message: "Hello",
            wantErr: false,
        },
        {
            name:    "invalid phone",
            phone:   "invalid",
            message: "Hello",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := SendTextMessage(context.Background(), tt.phone, tt.message)
            if (err != nil) != tt.wantErr {
                t.Errorf("SendTextMessage() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Coverage

- Aim for **80%+ test coverage** for new code
- All bug fixes should include regression tests
- Critical paths must have comprehensive test coverage

### Running Tests

```bash
# All tests
cd src && go test ./...

# With coverage
cd src && go test -cover ./...

# Coverage report
cd src && go test -coverprofile=coverage.out ./...
cd src && go tool cover -html=coverage.out
```

## Commit Messages

Follow **Conventional Commits** format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation changes
- `style` - Code style changes (formatting, no logic change)
- `refactor` - Code refactoring
- `test` - Adding or updating tests
- `chore` - Maintenance tasks, dependencies

### Scope

The scope should be the affected module:

- `send` - Send operations
- `group` - Group management
- `message` - Message handling
- `app` - Application lifecycle
- `webhook` - Webhook system
- `api` - API endpoints
- `mcp` - MCP server
- `docs` - Documentation

### Examples

```bash
# Feature
feat(send): add message scheduling feature

# Bug fix
fix(webhook): correct HMAC signature validation

# Documentation
docs(api): update endpoint examples

# Refactoring
refactor(send): simplify media upload logic

# Chore
chore(deps): update whatsmeow to v0.0.85
```

### Important Rules

- Use present tense ("add feature" not "added feature")
- Use imperative mood ("move cursor" not "moves cursor")
- Don't capitalize first letter
- No period at the end
- Keep description under 72 characters
- **No attribution** (e.g., no "Co-Authored-By: Claude")

## Pull Request Process

### Before Submitting

1. **Update your branch** with latest main:

```bash
git checkout main
git pull upstream main
git checkout username/feature-name
git rebase main
```

2. **Run all checks**:

```bash
cd src && go fmt ./...
cd src && go vet ./...
cd src && go test ./...
```

3. **Update documentation** if needed

4. **Self-review your changes**

### PR Title

Follow the same format as commit messages:

```
feat(send): add message scheduling feature
```

### PR Description

Use this template:

```markdown
## Description
Brief description of what this PR does.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Added unit tests
- [ ] Tested manually
- [ ] All tests passing

## Checklist
- [ ] My code follows the project's coding standards
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
```

### Review Process

1. **Automated Checks**: CI/CD will run tests and linters
2. **Code Review**: Maintainers will review your code
3. **Feedback**: Address any requested changes
4. **Approval**: Once approved, maintainers will merge

### After Merge

1. **Delete your branch**:

```bash
git branch -d username/feature-name
git push origin --delete username/feature-name
```

2. **Update your main branch**:

```bash
git checkout main
git pull upstream main
```

## Documentation

### When to Update Documentation

- Adding new features → Update relevant guides and API reference
- Changing APIs → Update OpenAPI spec and examples
- Bug fixes → Update troubleshooting guide if applicable
- Configuration changes → Update configuration reference

### Documentation Structure

```
docs/
├── getting-started/   # Tutorials for new users
├── guides/            # How-to guides for specific tasks
├── reference/         # Technical reference (API specs, config)
├── developer/         # Architecture, contributing, ADRs
├── operations/        # Deployment, monitoring, security
└── postmortems/       # Lessons learned from incidents
```

### Documentation Standards

- Use Markdown formatting
- Include code examples where applicable
- Add links to related documentation
- Keep language clear and concise
- Test all command examples

## Project Structure

Understanding the architecture helps with contributions. See [Architecture Overview](architecture.md) for details.

```
src/
├── cmd/               # CLI entry points (cobra commands)
├── config/            # Configuration management
├── domains/           # Domain logic (DDD)
│   ├── app/          # App lifecycle
│   ├── send/         # Send operations
│   ├── message/      # Message handling
│   └── ...
├── infrastructure/    # External integrations
│   ├── whatsapp/     # WhatsApp client
│   └── chatstorage/  # Storage
├── ui/                # User interfaces
│   ├── rest/         # REST API
│   ├── mcp/          # MCP server
│   └── websocket/    # WebSocket
├── usecase/           # Business logic orchestration
├── validations/       # Input validation
└── pkg/               # Shared utilities
```

## Common Tasks

### Adding a New API Endpoint

1. Define the request/response models in `domains/`
2. Implement the domain service in `domains/<module>/service_impl.go`
3. Add use case in `usecase/<module>.go`
4. Create the handler in `ui/rest/<module>.go`
5. Register the route in `ui/rest/routes.go`
6. Update OpenAPI specification in `docs/openapi.yaml`
7. Add tests
8. Update documentation

### Adding a New MCP Tool

1. Define the tool in `ui/mcp/tools.go`
2. Implement the handler
3. Register the tool in the MCP server
4. Add tests
5. Update MCP documentation

### Adding a New Configuration Option

1. Add to `config/settings.go`
2. Add to `cmd/root.go` (initEnvConfig)
3. Add to `src/.env.example`
4. Document in `docs/reference/configuration.md`

## Getting Help

- **Questions**: Open a [GitHub Discussion](https://github.com/aldinokemal/go-whatsapp-web-multidevice/discussions)
- **Bugs**: Open a [GitHub Issue](https://github.com/aldinokemal/go-whatsapp-web-multidevice/issues)
- **Security**: See [Security Policy](../../SECURITY.md) (if exists)

## Recognition

Contributors will be recognized in:
- GitHub contributors list
- Release notes (for significant contributions)

Thank you for contributing to go-whatsapp-web-multidevice!
