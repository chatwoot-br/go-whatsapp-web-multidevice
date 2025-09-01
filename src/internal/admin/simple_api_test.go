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
		mockLifecycle.On("UpdateInstanceConfig", 3001, mock.Anything).Return(expectedInstance, nil).Once()

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
		mockLifecycle.On("UpdateInstanceConfig", 3002, mock.Anything).Return(nil, fmt.Errorf("instance on port 3002 not found")).Once()

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
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
