# Testing Guide

This guide covers testing strategies and practices for go-whatsapp-web-multidevice.

## Table of Contents

- [Overview](#overview)
- [Running Tests](#running-tests)
- [Writing Tests](#writing-tests)
- [Test Coverage](#test-coverage)
- [Testing Strategies](#testing-strategies)
- [Continuous Integration](#continuous-integration)

## Overview

The project uses Go's built-in testing framework with table-driven tests as the primary pattern. The codebase uses `testify/assert` and `testify/mock` for assertions and mocking.

### Test Types

1. **Unit Tests** - Test individual functions and methods in isolation
2. **Integration Tests** - Test interactions between components
3. **E2E Tests** - Test complete workflows with real infrastructure

### Current Test Coverage

```bash
# Run this command to check current coverage
cd src && go test -cover ./...
```

**Coverage by Package** (as of latest run):
- `validations/` - 74.3% ✓ Good coverage
- `pkg/utils/` - 32.9% ⚠ Needs improvement
- `internal/admin/` - 35.6% ⚠ Needs improvement
- `usecase/` - 2.2% ⚠ Needs tests
- `infrastructure/whatsapp/` - 2.5% ⚠ Needs tests

Key areas with tests:
- `validations/` - Input validation logic (comprehensive)
- `usecase/` - Audio processing, document MIME detection
- `internal/admin/` - Admin API, mocks, integration tests
- `pkg/utils/` - Utility functions

Areas needing tests:
- Domain services (`domains/*/service_impl.go`)
- REST API handlers (`ui/rest/`)
- MCP tools (`ui/mcp/`)
- Infrastructure components (webhook, WhatsApp client)

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

The project uses `github.com/stretchr/testify/mock` for mocking dependencies. There are two approaches:

#### Approach 1: Manual Mock Implementation

For simple cases, create manual mocks:

```go
package mypackage

import (
    "context"
    domainSend "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/send"
)

// Mock implementation for tests
type mockSendService struct {
    mockSendText func(ctx context.Context, request domainSend.MessageRequest) (domainSend.MessageResponse, error)
}

func (m *mockSendService) SendText(ctx context.Context, request domainSend.MessageRequest) (domainSend.MessageResponse, error) {
    if m.mockSendText != nil {
        return m.mockSendText(ctx, request)
    }
    return domainSend.MessageResponse{}, nil
}

// Test using manual mock
func TestHandler_SendMessage(t *testing.T) {
    mockService := &mockSendService{
        mockSendText: func(ctx context.Context, request domainSend.MessageRequest) (domainSend.MessageResponse, error) {
            return domainSend.MessageResponse{
                MessageID: "test-msg-123",
                Status:    "success",
            }, nil
        },
    }

    // Use mockService in your handler test
    handler := &Send{Service: mockService}
    // ... test handler logic
}
```

#### Approach 2: testify/mock (Recommended)

For complex interfaces, use testify's mock package (see `internal/admin/mocks_test.go`):

```go
package admin

import (
    "github.com/stretchr/testify/mock"
)

// MockLifecycleManager implements ILifecycleManager
type MockLifecycleManager struct {
    mock.Mock
}

func (m *MockLifecycleManager) CreateInstance(port int) (*Instance, error) {
    args := m.Called(port)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Instance), args.Error(1)
}

func (m *MockLifecycleManager) GetInstance(port int) (*Instance, error) {
    args := m.Called(port)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Instance), args.Error(1)
}

// Test using testify mock
func TestCreateInstanceAPI(t *testing.T) {
    mockMgr := new(MockLifecycleManager)

    // Set expectations
    mockMgr.On("CreateInstance", 3000).Return(&Instance{
        Port: 3000,
        Status: "running",
    }, nil)

    // Use mockMgr in test
    result, err := mockMgr.CreateInstance(3000)

    // Assert expectations were met
    assert.NoError(t, err)
    assert.Equal(t, 3000, result.Port)
    mockMgr.AssertExpectations(t)
}
```

#### Best Practices for Mocking

1. **Mock at Interface Boundaries**: Mock external dependencies (database, HTTP clients, WhatsApp API)
2. **Don't Mock Everything**: Only mock what you need to isolate the code under test
3. **Use Dependency Injection**: Pass dependencies as interfaces to make testing easier
4. **Verify Expectations**: Use `AssertExpectations()` with testify mocks
5. **Keep Mocks Simple**: Don't add complex logic to mocks

### Testing HTTP Handlers

The REST API uses Fiber v2. Test handlers using `httptest` and mocked services:

```go
package rest

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http/httptest"
    "testing"

    domainSend "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/send"
    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
)

// Mock service for handler tests
type mockSendService struct {
    mockSendText func(ctx context.Context, request domainSend.MessageRequest) (domainSend.MessageResponse, error)
}

func (m *mockSendService) SendText(ctx context.Context, request domainSend.MessageRequest) (domainSend.MessageResponse, error) {
    if m.mockSendText != nil {
        return m.mockSendText(ctx, request)
    }
    return domainSend.MessageResponse{}, nil
}

func TestSendTextHandler(t *testing.T) {
    tests := []struct {
        name           string
        requestBody    domainSend.MessageRequest
        mockResponse   domainSend.MessageResponse
        mockError      error
        expectedStatus int
        expectedCode   string
    }{
        {
            name: "successful message send",
            requestBody: domainSend.MessageRequest{
                BaseRequest: domainSend.BaseRequest{
                    Phone: "6281234567890@s.whatsapp.net",
                },
                Message: "Hello, World!",
            },
            mockResponse: domainSend.MessageResponse{
                MessageID: "msg-123",
                Status:    "success",
            },
            mockError:      nil,
            expectedStatus: 200,
            expectedCode:   "SUCCESS",
        },
        {
            name: "empty phone number",
            requestBody: domainSend.MessageRequest{
                BaseRequest: domainSend.BaseRequest{
                    Phone: "",
                },
                Message: "Hello, World!",
            },
            mockError:      pkgError.ValidationError("phone: cannot be blank."),
            expectedStatus: 400,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mock service
            mockService := &mockSendService{
                mockSendText: func(ctx context.Context, request domainSend.MessageRequest) (domainSend.MessageResponse, error) {
                    return tt.mockResponse, tt.mockError
                },
            }

            // Create Fiber app and register handler
            app := fiber.New(fiber.Config{
                ErrorHandler: func(c *fiber.Ctx, err error) error {
                    code := fiber.StatusInternalServerError
                    if e, ok := err.(*fiber.Error); ok {
                        code = e.Code
                    }
                    return c.Status(code).JSON(fiber.Map{
                        "error": err.Error(),
                    })
                },
            })

            handler := &Send{Service: mockService}
            app.Post("/send/message", handler.SendText)

            // Create request
            body, _ := json.Marshal(tt.requestBody)
            req := httptest.NewRequest("POST", "/send/message", bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")

            // Execute request
            resp, err := app.Test(req)
            assert.NoError(t, err)
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)

            // Parse response
            var result map[string]interface{}
            json.NewDecoder(resp.Body).Decode(&result)

            if tt.expectedCode != "" {
                assert.Equal(t, tt.expectedCode, result["code"])
            }
        })
    }
}
```

#### Testing File Upload Handlers

For handlers with file uploads (images, videos, documents):

```go
func TestSendImageHandler(t *testing.T) {
    // Create mock multipart form
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    // Add form fields
    writer.WriteField("phone", "6281234567890@s.whatsapp.net")
    writer.WriteField("caption", "Test image")

    // Add file
    part, _ := writer.CreateFormFile("image", "test.jpg")
    part.Write([]byte("fake image data"))
    writer.Close()

    // Setup mock service
    mockService := &mockSendService{
        mockSendImage: func(ctx context.Context, request domainSend.ImageRequest) (domainSend.ImageResponse, error) {
            return domainSend.ImageResponse{
                MessageID: "img-123",
                Status:    "success",
            }, nil
        },
    }

    // Create Fiber app
    app := fiber.New()
    handler := &Send{Service: mockService}
    app.Post("/send/image", handler.SendImage)

    // Create request
    req := httptest.NewRequest("POST", "/send/image", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())

    // Execute and assert
    resp, err := app.Test(req)
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
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

Focus: Component interactions with real infrastructure (database, file system).

**Guidelines**:
- Test multiple components together
- Use real database (test database)
- Test realistic scenarios
- Clean up after tests
- Use temporary directories for file operations

**Example from internal/admin/integration_test.go**:

```go
package admin

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestCompleteConfigurationIntegration(t *testing.T) {
    // Create temporary directory for test
    tempDir, err := os.MkdirTemp("", "admin_integration_test")
    assert.NoError(t, err)
    defer os.RemoveAll(tempDir)

    // Set up environment variables
    envVars := map[string]string{
        "SUPERVISOR_CONF_DIR": filepath.Join(tempDir, "conf"),
        "INSTANCES_DIR":       filepath.Join(tempDir, "instances"),
        "SUPERVISOR_LOG_DIR":  filepath.Join(tempDir, "logs"),
        "GOWA_BIN":            "/usr/local/bin/whatsapp",
        "GOWA_BASIC_AUTH":     "admin:password123",
        "GOWA_DEBUG":          "true",
    }

    // Store original values
    originalValues := make(map[string]string)
    for key, value := range envVars {
        originalValues[key] = os.Getenv(key)
        os.Setenv(key, value)
    }

    // Restore original environment after test
    defer func() {
        for key, originalValue := range originalValues {
            if originalValue == "" {
                os.Unsetenv(key)
            } else {
                os.Setenv(key, originalValue)
            }
        }
    }()

    // Create configuration and writer
    config := DefaultInstanceConfig()
    writer, err := NewConfigWriter(config)
    assert.NoError(t, err)

    // Write configuration for port 3001
    port := 3001
    err = writer.WriteConfig(port)
    assert.NoError(t, err)

    // Verify the configuration file was created
    configPath := filepath.Join(config.ConfDir, "gowa-3001.conf")
    assert.FileExists(t, configPath)

    // Read and verify content
    content, err := os.ReadFile(configPath)
    assert.NoError(t, err)
    configStr := string(content)

    // Verify specific configurations
    assert.Contains(t, configStr, "[program:gowa_3001]")
    assert.Contains(t, configStr, "--port=3001")
    assert.Contains(t, configStr, "--debug=true")
    assert.Contains(t, configStr, "--basic-auth=admin:password123")

    // Verify directory creation
    assert.DirExists(t, config.ConfDir)
    assert.DirExists(t, config.LogDir)
    assert.DirExists(t, filepath.Join(config.InstancesDir, "3001", "storages"))
}
```

#### Integration Test Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Always clean up resources (temp files, database records)
3. **Realistic Data**: Use realistic test data
4. **Environment**: Set up and tear down test environment
5. **Temporary Resources**: Use `os.MkdirTemp()` for file operations

#### Testing Database Operations

For components that interact with the database:

```go
func TestDatabaseIntegration(t *testing.T) {
    // Create temporary database
    tempDB := filepath.Join(os.TempDir(), "test_whatsapp.db")
    defer os.Remove(tempDB)

    // Initialize database connection
    dbURI := fmt.Sprintf("file:%s?_foreign_keys=on", tempDB)
    db, err := sql.Open("sqlite3", dbURI)
    assert.NoError(t, err)
    defer db.Close()

    // Run migrations or create tables
    // ... setup database schema

    // Test database operations
    // ... your test logic

    // Cleanup
    db.Close()
    os.Remove(tempDB)
}
```

### E2E Testing

Focus: Complete workflows with real infrastructure (see `internal/admin/e2e_api_test.go`).

**E2E tests** verify entire user workflows from API request to response, including:
- HTTP routing
- Request parsing
- Business logic
- File system operations
- Response generation

#### Example E2E Test Structure

```go
package admin

import (
    "bytes"
    "encoding/json"
    "net/http/httptest"
    "os"
    "path/filepath"
    "testing"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
)

func TestAdminAPI_EndToEnd_ConfigGeneration(t *testing.T) {
    // Setup: Create temporary environment
    tempDir, err := os.MkdirTemp("", "admin_e2e_test")
    assert.NoError(t, err)
    defer os.RemoveAll(tempDir)

    // Configure environment
    os.Setenv("SUPERVISOR_CONF_DIR", filepath.Join(tempDir, "conf"))
    os.Setenv("INSTANCES_DIR", filepath.Join(tempDir, "instances"))
    os.Setenv("ADMIN_TOKEN", "test-token")
    defer os.Unsetenv("SUPERVISOR_CONF_DIR")
    defer os.Unsetenv("INSTANCES_DIR")
    defer os.Unsetenv("ADMIN_TOKEN")

    // Create Fiber app with real handlers
    app := fiber.New()

    app.Post("/admin/instances", func(c *fiber.Ctx) error {
        // Real handler implementation
        var req CreateInstanceRequest
        if err := c.BodyParser(&req); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
        }

        // Perform actual operations
        config := DefaultInstanceConfig()
        config.Port = req.Port

        writer, err := NewConfigWriter(config)
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": "failed to create writer"})
        }

        if err := writer.WriteConfig(req.Port); err != nil {
            return c.Status(500).JSON(fiber.Map{"error": "failed to write config"})
        }

        return c.Status(201).JSON(fiber.Map{
            "message": "Instance created",
            "port":    req.Port,
        })
    })

    // Test: Create instance with custom config
    t.Run("create instance with custom config", func(t *testing.T) {
        reqBody := CreateInstanceRequest{
            Port:      3001,
            BasicAuth: "custom:password",
            Debug:     boolPtr(true),
        }
        bodyBytes, _ := json.Marshal(reqBody)

        req := httptest.NewRequest("POST", "/admin/instances", bytes.NewReader(bodyBytes))
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Authorization", "Bearer test-token")

        resp, err := app.Test(req)
        assert.NoError(t, err)
        assert.Equal(t, 201, resp.StatusCode)

        // Verify: Check file system side effects
        configPath := filepath.Join(tempDir, "conf", "gowa-3001.conf")
        assert.FileExists(t, configPath)

        content, err := os.ReadFile(configPath)
        assert.NoError(t, err)
        assert.Contains(t, string(content), "--port=3001")
        assert.Contains(t, string(content), "--debug=true")
        assert.Contains(t, string(content), "--basic-auth=custom:password")

        // Verify: Check directory structure
        storageDir := filepath.Join(tempDir, "instances", "3001", "storages")
        assert.DirExists(t, storageDir)
    })
}
```

#### E2E Test with Docker (testcontainers)

For testing with external dependencies (PostgreSQL, Redis):

```go
package integration_test

import (
    "context"
    "testing"
    "time"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

func TestWhatsAppWithPostgres(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    ctx := context.Background()

    // Start PostgreSQL container
    pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "postgres:15-alpine",
            ExposedPorts: []string{"5432/tcp"},
            Env: map[string]string{
                "POSTGRES_PASSWORD": "testpass",
                "POSTGRES_DB":       "testdb",
            },
            WaitingFor: wait.ForLog("database system is ready to accept connections").
                WithStartupTimeout(30 * time.Second),
        },
        Started: true,
    })
    if err != nil {
        t.Fatal(err)
    }
    defer pgContainer.Terminate(ctx)

    // Get container host and port
    host, _ := pgContainer.Host(ctx)
    port, _ := pgContainer.MappedPort(ctx, "5432")

    // Configure application to use test database
    dbURI := fmt.Sprintf("postgres://postgres:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())

    // Run your E2E tests with real database
    // ... test logic
}
```

#### E2E Test Scenarios

Implement tests for these workflows:

1. **Authentication Flow**:
   - QR code generation → Scan → Connected
   - Pairing code → Enter code → Connected

2. **Message Sending Flow**:
   - Send text → Receive confirmation
   - Send image → Upload → Compress → Send → Verify
   - Send file → Upload → Send → Verify

3. **Group Management Flow**:
   - Create group → Add participants → Send message
   - Update group settings → Verify changes

4. **Webhook Flow**:
   - Configure webhook → Receive message → Webhook called
   - Verify HMAC signature

#### Running E2E Tests

```bash
# Run all tests including E2E
cd src && go test ./... -v

# Run only E2E tests (by convention, name them *_e2e_test.go)
cd src && go test ./... -v -run E2E

# Skip E2E tests (they're usually slower)
cd src && go test ./... -short
```

### Performance Testing

Performance testing ensures the application performs well under load and identifies bottlenecks.

#### Benchmark Tests

Go's built-in benchmarking tool is used for performance testing:

```go
package usecase

import (
    "context"
    "testing"
)

// Benchmark audio processing
func BenchmarkProcessAudioForWhatsApp(b *testing.B) {
    service := serviceSend{}
    audioData := make([]byte, 1024*1024) // 1MB audio file
    mimeType := "audio/wav"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _, _, _ = service.processAudioForWhatsApp(audioData, mimeType)
    }
}

// Benchmark message validation
func BenchmarkValidateSendMessage(b *testing.B) {
    ctx := context.Background()
    request := domainSend.MessageRequest{
        BaseRequest: domainSend.BaseRequest{
            Phone: "6281234567890@s.whatsapp.net",
        },
        Message: "Hello, World!",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = validations.ValidateSendMessage(ctx, request)
    }
}

// Benchmark with different input sizes
func BenchmarkImageCompression(b *testing.B) {
    sizes := []int{100 * 1024, 1024 * 1024, 5 * 1024 * 1024} // 100KB, 1MB, 5MB

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size_%dKB", size/1024), func(b *testing.B) {
            imageData := make([]byte, size)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = compressImage(imageData)
            }
        })
    }
}

// Benchmark memory allocations
func BenchmarkJSONEncoding(b *testing.B) {
    response := domainSend.MessageResponse{
        MessageID: "msg-123456789",
        Status:    "success",
    }

    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = json.Marshal(response)
    }
}
```

#### Running Benchmarks

```bash
# Run all benchmarks
cd src && go test -bench=. ./...

# Run specific benchmark
cd src && go test -bench=BenchmarkProcessAudio ./usecase

# Run with memory allocation stats
cd src && go test -bench=. -benchmem ./...

# Run with CPU profiling
cd src && go test -bench=. -cpuprofile=cpu.prof ./...

# Run with memory profiling
cd src && go test -bench=. -memprofile=mem.prof ./...

# Analyze CPU profile
go tool pprof cpu.prof
```

#### Load Testing with k6

For API load testing, use [k6](https://k6.io/):

```javascript
// load_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '30s', target: 20 },  // Ramp up to 20 users
        { duration: '1m', target: 20 },   // Stay at 20 users
        { duration: '30s', target: 0 },   // Ramp down to 0 users
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
        http_req_failed: ['rate<0.01'],   // Error rate should be less than 1%
    },
};

export default function () {
    const url = 'http://localhost:3000/send/message';
    const payload = JSON.stringify({
        phone: '6281234567890@s.whatsapp.net',
        message: 'Load test message',
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Basic YWRtaW46cGFzc3dvcmQ=',
        },
    };

    const res = http.post(url, payload, params);

    check(res, {
        'status is 200': (r) => r.status === 200,
        'response has message_id': (r) => JSON.parse(r.body).results.message_id !== undefined,
    });

    sleep(1);
}
```

Run k6 test:

```bash
# Install k6 (macOS)
brew install k6

# Run load test
k6 run load_test.js

# Run with more virtual users
k6 run --vus 50 --duration 2m load_test.js
```

#### Performance Testing Best Practices

1. **Baseline**: Establish performance baselines for critical operations
2. **Realistic Data**: Use realistic data sizes and scenarios
3. **Isolate**: Test one component at a time
4. **Monitor**: Track CPU, memory, and disk I/O during tests
5. **Profile**: Use Go's profiling tools to identify bottlenecks
6. **Repeat**: Run benchmarks multiple times to ensure consistency

#### Areas to Benchmark

- **Message sending throughput**: Messages per second
- **Media processing time**: Image/video compression duration
- **Database query performance**: Query execution time
- **Concurrent request handling**: Response time under load
- **Webhook delivery**: Time to deliver webhook events
- **Memory usage**: Heap allocations and garbage collection

## Continuous Integration

### GitHub Actions

The project uses GitHub Actions for CI/CD. Current workflows:

#### 1. Docker Image Build (.github/workflows/build-docker-image.yaml)

Triggered on:
- Git tags matching `v[0-9]+.[0-9]+.[0-9]+` (e.g., v7.10.1)
- Manual workflow dispatch

What it does:
- Creates Hetzner Cloud runners (AMD64 and ARM64)
- Builds Docker images for both architectures
- Pushes images to GitHub Container Registry (ghcr.io)
- Creates multi-arch manifest
- Tags: `latest`, `latest-amd`, `latest-arm`, `{version}`, `{version}-amd`, `{version}-arm`

```yaml
# Key steps:
- Checkout code
- Login to GitHub Container Registry
- Build Docker image (golang.Dockerfile)
- Push to ghcr.io
- Create multi-arch manifest
```

#### 2. Helm Chart Release (.github/workflows/chart-releaser.yaml)

Triggered on push to main branch with chart changes.

What it does:
- Packages Helm chart
- Creates GitHub release for chart
- Updates chart repository

### Recommended Test Workflow (Not Yet Implemented)

Create `.github/workflows/test.yaml`:

```yaml
name: Test

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.21'
        cache-dependency-path: src/go.sum

    - name: Install dependencies
      working-directory: ./src
      run: |
        go mod download
        sudo apt-get update
        sudo apt-get install -y ffmpeg

    - name: Run tests
      working-directory: ./src
      run: go test -v -race -coverprofile=coverage.out ./...

    - name: Run go vet
      working-directory: ./src
      run: go vet ./...

    - name: Run go fmt check
      working-directory: ./src
      run: |
        if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
          echo "Please run 'go fmt ./...'"
          gofmt -s -d .
          exit 1
        fi

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./src/coverage.out
        flags: unittests
        name: codecov-umbrella

    - name: Check coverage threshold
      working-directory: ./src
      run: |
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        threshold=70
        if (( $(echo "$coverage < $threshold" | bc -l) )); then
          echo "Coverage $coverage% is below threshold $threshold%"
          exit 1
        fi

  lint:
    name: Lint
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.21'

    - name: golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
        working-directory: ./src
        args: --timeout=5m

  build:
    name: Build
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.21'

    - name: Build binary
      working-directory: ./src
      run: go build -v -o whatsapp .

    - name: Check binary
      working-directory: ./src
      run: ./whatsapp --version
```

### Setting Up CI for Tests

1. **Create workflow file**: `.github/workflows/test.yaml` (see above)
2. **Configure secrets**: Add required secrets in GitHub repository settings
3. **Set up branch protection**: Require tests to pass before merging
4. **Add status badges**: Add CI status badges to README.md

```markdown
![Tests](https://github.com/{owner}/{repo}/actions/workflows/test.yaml/badge.svg)
![Coverage](https://codecov.io/gh/{owner}/{repo}/branch/main/graph/badge.svg)
```

### Pre-commit Hooks

Pre-commit hooks run automatically before each commit to ensure code quality.

#### Setting Up Git Hooks

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# Pre-commit hook for go-whatsapp-web-multidevice
# Place this file at: .git/hooks/pre-commit
# Make it executable: chmod +x .git/hooks/pre-commit

set -e

echo "Running pre-commit checks..."

# Change to src directory
cd src

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. Check Go formatting
echo -e "${YELLOW}Checking Go formatting...${NC}"
UNFORMATTED=$(gofmt -l . | grep -v vendor || true)
if [ -n "$UNFORMATTED" ]; then
    echo -e "${RED}The following files need formatting:${NC}"
    echo "$UNFORMATTED"
    echo -e "${YELLOW}Run: cd src && go fmt ./...${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Go formatting check passed${NC}"

# 2. Run go vet
echo -e "${YELLOW}Running go vet...${NC}"
if ! go vet ./... 2>&1 | grep -v "^#"; then
    echo -e "${GREEN}✓ go vet passed${NC}"
else
    echo -e "${RED}✗ go vet failed${NC}"
    exit 1
fi

# 3. Run tests (fast tests only, skip E2E)
echo -e "${YELLOW}Running tests...${NC}"
if go test -short -timeout=30s ./...; then
    echo -e "${GREEN}✓ Tests passed${NC}"
else
    echo -e "${RED}✗ Tests failed${NC}"
    exit 1
fi

# 4. Check for common issues
echo -e "${YELLOW}Checking for common issues...${NC}"

# Check for TODO comments being added
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)
if [ -n "$STAGED_GO_FILES" ]; then
    for FILE in $STAGED_GO_FILES; do
        if git diff --cached "$FILE" | grep -E '^\+.*TODO' > /dev/null; then
            echo -e "${YELLOW}Warning: Adding TODO comments in $FILE${NC}"
        fi
    done
fi

# Check for debug statements
for FILE in $STAGED_GO_FILES; do
    if git diff --cached "$FILE" | grep -E '^\+.*(fmt\.Print|log\.Print|panic)' > /dev/null; then
        echo -e "${YELLOW}Warning: Debug statements found in $FILE${NC}"
    fi
done

echo -e "${GREEN}✓ All pre-commit checks passed${NC}"
exit 0
```

Make it executable:

```bash
chmod +x .git/hooks/pre-commit
```

#### Alternative: Using pre-commit Framework

Install [pre-commit](https://pre-commit.com/):

```bash
# macOS
brew install pre-commit

# Or using pip
pip install pre-commit
```

Create `.pre-commit-config.yaml` in project root:

```yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
        args: ['--maxkb=5000']
      - id: check-merge-conflict

  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.1
    hooks:
      - id: go-fmt
        args: [-w, -s]
      - id: go-vet
      - id: go-unit-tests
        args: [-short, -timeout=30s]
      - id: go-mod-tidy

  - repo: https://github.com/golangci/golangci-lint
    rev: v1.55.2
    hooks:
      - id: golangci-lint
        args: [--timeout=5m]
```

Install hooks:

```bash
pre-commit install
```

Run manually:

```bash
# Run on all files
pre-commit run --all-files

# Run on staged files only
pre-commit run
```

#### Skip Hooks (When Needed)

Sometimes you need to skip hooks (use sparingly):

```bash
# Skip all hooks
git commit --no-verify -m "Your message"

# Or set environment variable
SKIP=go-unit-tests git commit -m "Skip tests for this commit"
```

#### Recommended Checks

1. **Go formatting**: Ensure all Go files are properly formatted
2. **Go vet**: Static analysis to find bugs
3. **Unit tests**: Run fast unit tests (with `-short` flag)
4. **Import organization**: Ensure imports are organized
5. **No debug statements**: Check for fmt.Print, log.Print, panic
6. **File size limits**: Prevent committing large files
7. **Trailing whitespace**: Remove trailing whitespace

## Test Helpers

### Common Test Utilities

Create a test utility package for reusable test helpers. Place in `src/pkg/testutil/helpers.go`:

```go
package testutil

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
)

// CreateTestContext creates a context for testing with timeout
func CreateTestContext(t *testing.T) context.Context {
    t.Helper()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    t.Cleanup(cancel)
    return ctx
}

// CreateTempDir creates a temporary directory for tests
func CreateTempDir(t *testing.T, prefix string) string {
    t.Helper()
    dir, err := os.MkdirTemp("", prefix)
    if err != nil {
        t.Fatalf("failed to create temp dir: %v", err)
    }
    t.Cleanup(func() {
        os.RemoveAll(dir)
    })
    return dir
}

// CreateTempFile creates a temporary file with content
func CreateTempFile(t *testing.T, dir, pattern, content string) string {
    t.Helper()
    file, err := os.CreateTemp(dir, pattern)
    if err != nil {
        t.Fatalf("failed to create temp file: %v", err)
    }
    defer file.Close()

    if _, err := file.WriteString(content); err != nil {
        t.Fatalf("failed to write to temp file: %v", err)
    }

    return file.Name()
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
    t.Helper()
    if err == nil {
        if len(msgAndArgs) > 0 {
            t.Fatalf("expected error: %v", msgAndArgs[0])
        }
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

// AssertContains fails if haystack doesn't contain needle
func AssertContains(t *testing.T, haystack, needle string) {
    t.Helper()
    if !strings.Contains(haystack, needle) {
        t.Errorf("expected %q to contain %q", haystack, needle)
    }
}

// SetEnv sets environment variables for testing and restores them after
func SetEnv(t *testing.T, vars map[string]string) {
    t.Helper()
    original := make(map[string]string)

    for key, value := range vars {
        original[key] = os.Getenv(key)
        os.Setenv(key, value)
    }

    t.Cleanup(func() {
        for key, value := range original {
            if value == "" {
                os.Unsetenv(key)
            } else {
                os.Setenv(key, value)
            }
        }
    })
}

// WaitFor waits for a condition to be true or times out
func WaitFor(t *testing.T, condition func() bool, timeout time.Duration, message string) {
    t.Helper()
    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        if condition() {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }

    t.Fatalf("timeout waiting for: %s", message)
}
```

### Usage Examples

```go
package mypackage

import (
    "testing"
    "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/testutil"
)

func TestWithHelpers(t *testing.T) {
    // Create test context with cleanup
    ctx := testutil.CreateTestContext(t)

    // Create temporary directory
    tempDir := testutil.CreateTempDir(t, "mytest")

    // Set environment variables
    testutil.SetEnv(t, map[string]string{
        "APP_PORT": "3000",
        "APP_DEBUG": "true",
    })

    // Create temporary file
    configFile := testutil.CreateTempFile(t, tempDir, "config-*.yaml", "test: true")

    // Test assertions
    result, err := DoSomething(ctx, configFile)
    testutil.AssertNoError(t, err)
    testutil.AssertEqual(t, result.Status, "success")
    testutil.AssertContains(t, result.Message, "completed")

    // Wait for async operation
    testutil.WaitFor(t, func() bool {
        return result.IsComplete()
    }, 5*time.Second, "operation to complete")
}
```

### Test Fixtures

For managing test data, create a fixtures package:

```go
// src/pkg/testutil/fixtures.go
package testutil

import (
    domainSend "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/send"
)

// ValidSendRequest returns a valid send request for testing
func ValidSendRequest() domainSend.MessageRequest {
    return domainSend.MessageRequest{
        BaseRequest: domainSend.BaseRequest{
            Phone: "6281234567890@s.whatsapp.net",
        },
        Message: "Test message",
    }
}

// ValidPhoneNumber returns a valid phone number in JID format
func ValidPhoneNumber() string {
    return "6281234567890@s.whatsapp.net"
}

// ValidGroupJID returns a valid group JID
func ValidGroupJID() string {
    return "120363123456789012@g.us"
}

// CreateMockImageFile creates a minimal valid image file for testing
func CreateMockImageFile(t *testing.T) *multipart.FileHeader {
    t.Helper()

    // Create minimal PNG
    pngData := []byte{
        0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
        // ... minimal PNG data
    }

    return &multipart.FileHeader{
        Filename: "test.png",
        Size:     int64(len(pngData)),
        Header:   map[string][]string{"Content-Type": {"image/png"}},
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
- [CI/CD Workflows](../../.github/workflows/) - GitHub Actions pipelines
  - `build-docker-image.yaml` - Builds and pushes Docker images for AMD64/ARM64 on version tags
  - `chart-releaser.yaml` - Releases Helm charts on version tags

## Contributing to Tests

While this testing guide is comprehensive, the actual test coverage needs improvement. Contributions welcome:

- **High Priority**:
  - Add tests for domain services (`domains/*/service_impl.go`)
  - Add tests for REST API handlers (`ui/rest/`)
  - Add tests for MCP tools (`ui/mcp/`)

- **Medium Priority**:
  - Improve coverage in `usecase/` layer
  - Add integration tests for infrastructure layer
  - Create E2E test suite with testcontainers

- **Nice to Have**:
  - Performance/load testing suite
  - Contract testing for APIs
  - Chaos/resilience testing

See [Contributing Guide](contributing.md) for how to contribute.

## Resources

- [Go Testing](https://golang.org/pkg/testing/) - Official documentation
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests) - Best practices
- [Go Test Coverage](https://blog.golang.org/cover) - Coverage tools
- [Testing in Go](https://quii.gitbook.io/learn-go-with-tests/) - Learn by example
