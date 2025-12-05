package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function to create test API
func setupTestAPI(t *testing.T) (*AdminAPI, *MockLifecycleManager) {
	os.Setenv("ADMIN_TOKEN", "test-token")
	t.Cleanup(func() { os.Unsetenv("ADMIN_TOKEN") })

	mockLifecycle := &MockLifecycleManager{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	api, err := NewAdminAPI(mockLifecycle, logger)
	assert.NoError(t, err)

	return api, mockLifecycle
}

func TestNewAdminAPI(t *testing.T) {
	t.Run("returns error when ADMIN_TOKEN is not set", func(t *testing.T) {
		os.Unsetenv("ADMIN_TOKEN")
		mockLifecycle := &MockLifecycleManager{}
		logger := logrus.New()

		api, err := NewAdminAPI(mockLifecycle, logger)
		assert.Error(t, err)
		assert.Nil(t, api)
		assert.Contains(t, err.Error(), "ADMIN_TOKEN")
	})

	t.Run("creates API when ADMIN_TOKEN is set", func(t *testing.T) {
		os.Setenv("ADMIN_TOKEN", "test-token")
		defer os.Unsetenv("ADMIN_TOKEN")

		mockLifecycle := &MockLifecycleManager{}
		logger := logrus.New()

		api, err := NewAdminAPI(mockLifecycle, logger)
		assert.NoError(t, err)
		assert.NotNil(t, api)
	})
}

func TestAdminAPI_Authentication(t *testing.T) {
	api, _ := setupTestAPI(t)

	t.Run("returns 401 when Authorization header is missing", func(t *testing.T) {
		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/admin/instances", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("returns 401 when Authorization is not Bearer", func(t *testing.T) {
		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/admin/instances", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("returns 401 when Bearer token is invalid", func(t *testing.T) {
		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/admin/instances", nil)
		req.Header.Set("Authorization", "Bearer wrong-token")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAdminAPI_CreateInstance(t *testing.T) {
	api, mockLifecycle := setupTestAPI(t)

	expectedInstance := &Instance{
		Port:  3001,
		State: StateRunning,
	}

	t.Run("minimal config calls CreateInstance", func(t *testing.T) {
		mockLifecycle.On("CreateInstance", mock.Anything, 3001).Return(expectedInstance, nil).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := CreateInstanceRequest{Port: 3001}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/admin/instances", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("custom config calls CreateInstanceWithConfig", func(t *testing.T) {
		mockLifecycle.On("CreateInstanceWithConfig", mock.Anything, 3002, mock.Anything).Return(expectedInstance, nil).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := CreateInstanceRequest{
			Port:      3002,
			BasicAuth: "custom:auth",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/admin/instances", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 400 for invalid port range", func(t *testing.T) {
		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := CreateInstanceRequest{Port: 80} // Too low
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/admin/instances", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("POST", "/admin/instances", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 409 when instance already exists", func(t *testing.T) {
		mockLifecycle.On("CreateInstance", mock.Anything, 3003).Return(nil, errors.New("instance already exists")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := CreateInstanceRequest{Port: 3003}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/admin/instances", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 409 when port is in use", func(t *testing.T) {
		mockLifecycle.On("CreateInstance", mock.Anything, 3004).Return(nil, errors.New("port 3004 is already in use")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := CreateInstanceRequest{Port: 3004}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/admin/instances", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 409 when port is locked", func(t *testing.T) {
		mockLifecycle.On("CreateInstance", mock.Anything, 3005).Return(nil, errors.New("port locked by another operation")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := CreateInstanceRequest{Port: 3005}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/admin/instances", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 502 when supervisor unreachable", func(t *testing.T) {
		mockLifecycle.On("CreateInstance", mock.Anything, 3006).Return(nil, errors.New("connection refused")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := CreateInstanceRequest{Port: 3006}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/admin/instances", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})
}

func TestAdminAPI_ListInstances(t *testing.T) {
	api, mockLifecycle := setupTestAPI(t)

	t.Run("returns list of instances", func(t *testing.T) {
		instances := []*Instance{
			{Port: 3001, State: StateRunning},
			{Port: 3002, State: StateStopped},
		}
		mockLifecycle.On("ListInstances", mock.Anything).Return(instances, nil).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/admin/instances", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 502 when supervisor is unreachable", func(t *testing.T) {
		mockLifecycle.On("ListInstances", mock.Anything).Return(nil, errors.New("connection refused")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/admin/instances", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})
}

func TestAdminAPI_GetInstance(t *testing.T) {
	api, mockLifecycle := setupTestAPI(t)

	t.Run("returns instance details", func(t *testing.T) {
		instance := &Instance{Port: 3001, State: StateRunning}
		mockLifecycle.On("GetInstance", mock.Anything, 3001).Return(instance, nil).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/admin/instances/3001", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 404 when instance not found", func(t *testing.T) {
		mockLifecycle.On("GetInstance", mock.Anything, 9999).Return(nil, errors.New("not found")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/admin/instances/9999", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 400 for invalid port", func(t *testing.T) {
		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/admin/instances/invalid", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 502 when supervisor unreachable", func(t *testing.T) {
		mockLifecycle.On("GetInstance", mock.Anything, 9998).Return(nil, errors.New("connection refused")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/admin/instances/9998", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})
}

func TestAdminAPI_DeleteInstance(t *testing.T) {
	api, mockLifecycle := setupTestAPI(t)

	t.Run("deletes instance successfully", func(t *testing.T) {
		mockLifecycle.On("DeleteInstance", mock.Anything, 3001).Return(nil).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("DELETE", "/admin/instances/3001", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 404 when instance not found", func(t *testing.T) {
		mockLifecycle.On("DeleteInstance", mock.Anything, 9999).Return(errors.New("not found")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("DELETE", "/admin/instances/9999", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 409 when port is locked", func(t *testing.T) {
		mockLifecycle.On("DeleteInstance", mock.Anything, 3002).Return(errors.New("port locked by another operation")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("DELETE", "/admin/instances/3002", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("returns 502 when supervisor unreachable", func(t *testing.T) {
		mockLifecycle.On("DeleteInstance", mock.Anything, 3003).Return(errors.New("connection refused")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("DELETE", "/admin/instances/3003", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})
}

func TestAdminAPI_HealthEndpoints(t *testing.T) {
	api, mockLifecycle := setupTestAPI(t)

	t.Run("healthz returns healthy when supervisor is healthy", func(t *testing.T) {
		mockLifecycle.On("IsHealthy").Return(true).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/healthz", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("healthz returns degraded when supervisor is unhealthy", func(t *testing.T) {
		mockLifecycle.On("IsHealthy").Return(false).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/healthz", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("readyz returns 200 when supervisor is reachable", func(t *testing.T) {
		mockLifecycle.On("Ping").Return(nil).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/readyz", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	t.Run("readyz returns 503 when supervisor is unreachable", func(t *testing.T) {
		mockLifecycle.On("Ping").Return(errors.New("connection refused")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("GET", "/readyz", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})
}

func TestClassifySupervisorError(t *testing.T) {
	api, _ := setupTestAPI(t)

	tests := []struct {
		name         string
		errMsg       string
		expectedCode int
		expectedErr  string
	}{
		{"connection refused", "connection refused", http.StatusBadGateway, "supervisor_unreachable"},
		{"no such host", "no such host", http.StatusBadGateway, "supervisor_unreachable"},
		{"failed to ping", "failed to ping supervisor", http.StatusBadGateway, "supervisor_unreachable"},
		{"timeout", "context deadline exceeded: timeout", http.StatusGatewayTimeout, "supervisor_timeout"},
		{"deadline exceeded", "deadline exceeded", http.StatusGatewayTimeout, "supervisor_timeout"},
		{"other error", "some other error", http.StatusInternalServerError, "supervisor_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, errCode := api.classifySupervisorError(errors.New(tt.errMsg))
			assert.Equal(t, tt.expectedCode, status)
			assert.Equal(t, tt.expectedErr, errCode)
		})
	}
}

func TestHasCustomConfig(t *testing.T) {
	api, _ := setupTestAPI(t)

	t.Run("returns false for empty request", func(t *testing.T) {
		req := CreateInstanceRequest{Port: 3001}
		assert.False(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when BasicAuth is set", func(t *testing.T) {
		req := CreateInstanceRequest{Port: 3001, BasicAuth: "user:pass"}
		assert.True(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when Debug is set", func(t *testing.T) {
		debug := true
		req := CreateInstanceRequest{Port: 3001, Debug: &debug}
		assert.True(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when OS is set", func(t *testing.T) {
		req := CreateInstanceRequest{Port: 3001, OS: "Firefox"}
		assert.True(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when AccountValidation is set", func(t *testing.T) {
		val := true
		req := CreateInstanceRequest{Port: 3001, AccountValidation: &val}
		assert.True(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when BasePath is set", func(t *testing.T) {
		req := CreateInstanceRequest{Port: 3001, BasePath: "/api"}
		assert.True(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when AutoReply is set", func(t *testing.T) {
		req := CreateInstanceRequest{Port: 3001, AutoReply: "Hello"}
		assert.True(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when AutoMarkRead is set", func(t *testing.T) {
		val := true
		req := CreateInstanceRequest{Port: 3001, AutoMarkRead: &val}
		assert.True(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when Webhook is set", func(t *testing.T) {
		req := CreateInstanceRequest{Port: 3001, Webhook: "https://example.com/hook"}
		assert.True(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when WebhookSecret is set", func(t *testing.T) {
		req := CreateInstanceRequest{Port: 3001, WebhookSecret: "my-secret"}
		assert.True(t, api.hasCustomConfig(req))
	})

	t.Run("returns true when ChatStorage is set", func(t *testing.T) {
		val := false
		req := CreateInstanceRequest{Port: 3001, ChatStorage: &val}
		assert.True(t, api.hasCustomConfig(req))
	})
}

func TestBuildInstanceConfig(t *testing.T) {
	api, _ := setupTestAPI(t)

	t.Run("returns defaults for empty request", func(t *testing.T) {
		req := CreateInstanceRequest{Port: 3001}
		config := api.buildInstanceConfig(req)

		assert.NotNil(t, config)
		assert.Equal(t, "admin:admin", config.BasicAuth)
		assert.Equal(t, "Chrome", config.OS)
	})

	t.Run("overrides all fields when set", func(t *testing.T) {
		debug := true
		accountVal := true
		autoMarkRead := true
		chatStorage := false

		req := CreateInstanceRequest{
			Port:              3001,
			BasicAuth:         "custom:auth",
			Debug:             &debug,
			OS:                "Firefox",
			AccountValidation: &accountVal,
			BasePath:          "/api",
			AutoReply:         "Hello",
			AutoMarkRead:      &autoMarkRead,
			Webhook:           "https://example.com",
			WebhookSecret:     "secret123",
			ChatStorage:       &chatStorage,
		}
		config := api.buildInstanceConfig(req)

		assert.Equal(t, "custom:auth", config.BasicAuth)
		assert.Equal(t, true, config.Debug)
		assert.Equal(t, "Firefox", config.OS)
		assert.Equal(t, true, config.AccountValidation)
		assert.Equal(t, "/api", config.BasePath)
		assert.Equal(t, "Hello", config.AutoReply)
		assert.Equal(t, true, config.AutoMarkRead)
		assert.Equal(t, "https://example.com", config.Webhook)
		assert.Equal(t, "secret123", config.WebhookSecret)
		assert.Equal(t, false, config.ChatStorage)
	})
}

func TestBuildInstanceConfigFromUpdate(t *testing.T) {
	api, _ := setupTestAPI(t)

	t.Run("returns defaults with port for empty request", func(t *testing.T) {
		req := UpdateInstanceRequest{}
		config := api.buildInstanceConfigFromUpdate(3001, req)

		assert.NotNil(t, config)
		assert.Equal(t, 3001, config.Port)
		assert.Equal(t, "admin:admin", config.BasicAuth)
	})

	t.Run("overrides all fields when set", func(t *testing.T) {
		basicAuth := "custom:auth"
		debug := true
		os := "Firefox"
		accountVal := true
		basePath := "/api"
		autoReply := "Hello"
		autoMarkRead := true
		webhook := "https://example.com"
		webhookSecret := "secret123"
		chatStorage := false

		req := UpdateInstanceRequest{
			BasicAuth:         &basicAuth,
			Debug:             &debug,
			OS:                &os,
			AccountValidation: &accountVal,
			BasePath:          &basePath,
			AutoReply:         &autoReply,
			AutoMarkRead:      &autoMarkRead,
			Webhook:           &webhook,
			WebhookSecret:     &webhookSecret,
			ChatStorage:       &chatStorage,
		}
		config := api.buildInstanceConfigFromUpdate(3002, req)

		assert.Equal(t, 3002, config.Port)
		assert.Equal(t, "custom:auth", config.BasicAuth)
		assert.Equal(t, true, config.Debug)
		assert.Equal(t, "Firefox", config.OS)
		assert.Equal(t, true, config.AccountValidation)
		assert.Equal(t, "/api", config.BasePath)
		assert.Equal(t, "Hello", config.AutoReply)
		assert.Equal(t, true, config.AutoMarkRead)
		assert.Equal(t, "https://example.com", config.Webhook)
		assert.Equal(t, "secret123", config.WebhookSecret)
		assert.Equal(t, false, config.ChatStorage)
	})
}
