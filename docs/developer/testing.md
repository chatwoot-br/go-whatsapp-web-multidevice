# Testing Guide

> **Note**: This document is a work in progress. Contributions are welcome!

This guide covers testing strategies and practices for go-whatsapp-web-multidevice.

## Table of Contents

- [Overview](#overview)
- [Running Tests](#running-tests)
- [Writing Tests](#writing-tests)
- [Test Coverage](#test-coverage)
- [Testing Strategies](#testing-strategies)
- [Continuous Integration](#continuous-integration)

## Overview

The project uses Go's built-in testing framework with table-driven tests as the primary pattern.

### Test Types

1. **Unit Tests** - Test individual functions and methods in isolation
2. **Integration Tests** - Test interactions between components
3. **E2E Tests** - Test complete workflows (TODO)

### Current Test Coverage

> **TODO**: Add coverage badge and current coverage percentage

Key areas with tests:
- `validations/` - Input validation logic
- `usecase/` - Some use case logic (send_audio_test.go)

Areas needing tests:
- Domain services
- REST API handlers
- MCP tools
- Infrastructure components

## Running Tests

### All Tests

```bash
cd src
go test ./...
```

### Specific Package

```bash
cd src
go test ./validations
go test ./usecase
```

### With Coverage

```bash
cd src
go test -cover ./...
```

### Verbose Output

```bash
cd src
go test -v ./...
```

### Coverage Report

Generate HTML coverage report:

```bash
cd src
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

This opens a browser with line-by-line coverage visualization.

### Run Specific Test

```bash
cd src
go test -run TestSendAudio ./usecase
```

### With Race Detector

```bash
cd src
go test -race ./...
```

## Writing Tests

### Test File Naming

- Test files: `*_test.go`
- Place in same package as code being tested
- Example: `send.go` → `send_test.go`

### Test Function Naming

```go
func TestFunctionName(t *testing.T)
func TestStructName_MethodName(t *testing.T)
```

Examples:
- `TestValidatePhone`
- `TestSendUsecase_SendTextMessage`

### Table-Driven Tests

Preferred pattern for testing multiple scenarios:

```go
func TestValidatePhone(t *testing.T) {
    tests := []struct {
        name    string
        phone   string
        wantErr bool
    }{
        {
            name:    "valid phone with country code",
            phone:   "6281234567890",
            wantErr: false,
        },
        {
            name:    "invalid phone - too short",
            phone:   "123",
            wantErr: true,
        },
        {
            name:    "invalid phone - with plus",
            phone:   "+6281234567890",
            wantErr: true,
        },
        {
            name:    "invalid phone - with spaces",
            phone:   "62 812 3456 7890",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePhone(tt.phone)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidatePhone() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Testing with Mocks

> **TODO**: Add mocking examples

For testing usecases that depend on external services, use interfaces and mocks:

```go
// Interface (already defined in domains/)
type IUserService interface {
    GetUserInfo(ctx context.Context, phone string) (*User, error)
}

// Mock implementation for tests
type mockUserService struct {
    mockGetUserInfo func(ctx context.Context, phone string) (*User, error)
}

func (m *mockUserService) GetUserInfo(ctx context.Context, phone string) (*User, error) {
    if m.mockGetUserInfo != nil {
        return m.mockGetUserInfo(ctx, phone)
    }
    return nil, nil
}

// Test using mock
func TestSendMessage_WithUserValidation(t *testing.T) {
    mockUser := &mockUserService{
        mockGetUserInfo: func(ctx context.Context, phone string) (*User, error) {
            return &User{Phone: phone, Name: "Test User"}, nil
        },
    }

    // Use mockUser in test
}
```

### Testing HTTP Handlers

> **TODO**: Add REST API handler testing examples

```go
// Example structure (to be implemented)
func TestSendTextHandler(t *testing.T) {
    app := fiber.New()
    // Setup routes
    // Create test request
    req := httptest.NewRequest("POST", "/send/text", body)
    resp, err := app.Test(req)
    // Assert response
}
```

### Testing Errors

Always test both success and error cases:

```go
func TestSendMessage(t *testing.T) {
    tests := []struct {
        name    string
        input   SendRequest
        wantErr bool
        errMsg  string
    }{
        {
            name: "success",
            input: SendRequest{
                Phone:   "6281234567890",
                Message: "Hello",
            },
            wantErr: false,
        },
        {
            name: "empty phone",
            input: SendRequest{
                Phone:   "",
                Message: "Hello",
            },
            wantErr: true,
            errMsg:  "phone is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := SendMessage(tt.input)
            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                }
                if !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
                }
            } else {
                if err != nil {
                    t.Errorf("unexpected error: %v", err)
                }
            }
        })
    }
}
```

## Test Coverage

### Coverage Goals

- **New Code**: 80%+ coverage
- **Critical Paths**: 100% coverage
- **Bug Fixes**: Add regression tests

### Checking Coverage

```bash
# Overall coverage
cd src && go test -cover ./...

# Detailed coverage by package
cd src && go test -coverprofile=coverage.out ./...
cd src && go tool cover -func=coverage.out

# Coverage for specific package
cd src && go test -cover ./validations
```

### Coverage Report

```bash
cd src && go test -coverprofile=coverage.out ./...
cd src && go tool cover -html=coverage.out
```

### Excluding Files from Coverage

Some files may not need testing (e.g., generated code):

```bash
# Coverage excluding generated files
go test -cover ./... -coverpkg=./... | grep -v "_gen.go"
```

## Testing Strategies

### Unit Testing

Focus: Individual functions and methods

**Guidelines**:
- Test one thing at a time
- Use mocks for dependencies
- Test edge cases
- Test error conditions

**Example**: Validation functions

```bash
cd src && go test ./validations
```

### Integration Testing

Focus: Component interactions

**Guidelines**:
- Test multiple components together
- Use real database (test database)
- Test realistic scenarios
- Clean up after tests

**Example**: Usecase with domain services

> **TODO**: Add integration test examples

### E2E Testing

Focus: Complete workflows

> **TODO**: Implement E2E tests

Planned E2E tests:
- Login → Send message → Verify delivery
- Create group → Add participants → Send message
- Upload media → Download media → Verify content

### Performance Testing

> **TODO**: Add performance testing guidelines

Areas to benchmark:
- Message sending throughput
- Media processing time
- Database query performance
- Concurrent request handling

## Continuous Integration

### GitHub Actions

Current CI pipeline (from `.github/workflows/`):

> **TODO**: Document CI/CD pipeline

Planned CI checks:
- `go test ./...` - Run all tests
- `go vet ./...` - Static analysis
- `golint ./...` - Linting
- Coverage report
- Build verification

### Pre-commit Hooks

> **TODO**: Add pre-commit hook examples

Recommended pre-commit checks:

```bash
#!/bin/bash
# .git/hooks/pre-commit

cd src

# Run tests
go test ./...
if [ $? -ne 0 ]; then
    echo "Tests failed"
    exit 1
fi

# Format code
go fmt ./...

# Vet code
go vet ./...
if [ $? -ne 0 ]; then
    echo "go vet failed"
    exit 1
fi

exit 0
```

## Test Helpers

### Common Test Utilities

> **TODO**: Create test utility package

Suggested test helpers:

```go
// testutil/helpers.go

// CreateTestContext creates a context for testing
func CreateTestContext() context.Context {
    return context.Background()
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error) {
    t.Helper()
    if err == nil {
        t.Fatal("expected error, got nil")
    }
}

// AssertEqual fails the test if got != want
func AssertEqual(t *testing.T, got, want interface{}) {
    t.Helper()
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}
```

## Testing Best Practices

1. **Test Names**: Describe what is being tested
2. **Table-Driven**: Use for multiple scenarios
3. **Isolation**: Tests should not depend on each other
4. **Cleanup**: Always clean up resources
5. **Deterministic**: Tests should produce consistent results
6. **Fast**: Keep tests fast to encourage frequent running
7. **Readable**: Tests are documentation

## Debugging Tests

### Run with Verbose Output

```bash
cd src && go test -v ./...
```

### Print Debug Info

```go
func TestSomething(t *testing.T) {
    result := DoSomething()
    t.Logf("Result: %+v", result)  // Only shown with -v flag

    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

### Run Single Test

```bash
cd src && go test -v -run TestSpecificTest ./package
```

## Areas Needing Tests

Priority list for adding tests:

1. **High Priority**
   - Domain services (domains/*/service_impl.go)
   - REST API handlers (ui/rest/)
   - Input validation (validations/)

2. **Medium Priority**
   - Usecases (usecase/)
   - MCP tools (ui/mcp/)
   - Webhook handling

3. **Low Priority**
   - Configuration parsing
   - Utility functions

## Related Documentation

- [Contributing Guide](contributing.md) - How to contribute
- [Architecture Overview](architecture.md) - System design
- [CI/CD Documentation](../../.github/workflows/) - Pipeline details (TODO)

## Contributing

This testing guide is incomplete. Contributions needed:

- Add example tests for each layer
- Implement mocking patterns
- Create integration test suite
- Add E2E test framework
- Document CI/CD pipeline
- Add performance testing guide

See [Contributing Guide](contributing.md) for details.

## Resources

- [Go Testing](https://golang.org/pkg/testing/) - Official documentation
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests) - Best practices
- [Go Test Coverage](https://blog.golang.org/cover) - Coverage tools
- [Testing in Go](https://quii.gitbook.io/learn-go-with-tests/) - Learn by example
