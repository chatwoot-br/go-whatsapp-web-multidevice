package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAdminAPI_UpdateInstance(t *testing.T) {
	// Set required environment variable
	os.Setenv("ADMIN_TOKEN", "test-token")
	defer os.Unsetenv("ADMIN_TOKEN")

	mockLifecycle := &MockLifecycleManager{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	api, err := NewAdminAPI(mockLifecycle, logger)
	assert.NoError(t, err)

	expectedInstance := &Instance{
		Port:  3001,
		State: StateRunning,
	}

	// Test 1: Update instance with partial config
	t.Run("update instance with partial config", func(t *testing.T) {
		mockLifecycle.On("UpdateInstanceConfig", mock.Anything, 3001, mock.Anything).Return(expectedInstance, nil).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := UpdateInstanceRequest{
			Debug:   new(bool), // false
			Webhook: stringPtr("https://new-webhook.com"),
		}
		*reqBody.Debug = false
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/admin/instances/3001", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	// Test 2: Update instance - instance not found
	t.Run("update non-existent instance", func(t *testing.T) {
		mockLifecycle.On("UpdateInstanceConfig", mock.Anything, 3002, mock.Anything).Return(nil, fmt.Errorf("instance on port 3002 not found")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := UpdateInstanceRequest{
			Debug: new(bool),
		}
		*reqBody.Debug = true
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/admin/instances/3002", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	// Test 3: Update instance - invalid port
	t.Run("update instance with invalid port", func(t *testing.T) {
		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := UpdateInstanceRequest{
			Debug: new(bool),
		}
		*reqBody.Debug = true
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/admin/instances/invalid", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test 4: Update instance - invalid JSON
	t.Run("update instance with invalid JSON", func(t *testing.T) {
		app := fiber.New()
		api.SetupRoutes(app)

		req := httptest.NewRequest("PATCH", "/admin/instances/3001", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test 5: Update instance - port locked
	t.Run("update instance with port locked", func(t *testing.T) {
		mockLifecycle.On("UpdateInstanceConfig", mock.Anything, 3003, mock.Anything).Return(nil, fmt.Errorf("port locked by another operation")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := UpdateInstanceRequest{
			Debug: new(bool),
		}
		*reqBody.Debug = true
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/admin/instances/3003", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})

	// Test 6: Update instance - supervisor error
	t.Run("update instance with supervisor error", func(t *testing.T) {
		mockLifecycle.On("UpdateInstanceConfig", mock.Anything, 3004, mock.Anything).Return(nil, fmt.Errorf("connection refused")).Once()

		app := fiber.New()
		api.SetupRoutes(app)

		reqBody := UpdateInstanceRequest{
			Debug: new(bool),
		}
		*reqBody.Debug = true
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/admin/instances/3004", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)

		mockLifecycle.AssertExpectations(t)
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
