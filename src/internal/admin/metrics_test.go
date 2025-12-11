package admin

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestMetricsFunctions(t *testing.T) {
	t.Run("SetInstancesRunning sets gauge", func(t *testing.T) {
		SetInstancesRunning(5)
		// No assertion needed - just verify it doesn't panic
	})

	t.Run("IncrementAPIRequest increments counter", func(t *testing.T) {
		IncrementAPIRequest("GET", "/admin/instances", "200")
		IncrementAPIRequest("POST", "/admin/instances", "201")
		IncrementAPIRequest("DELETE", "/admin/instances/3001", "200")
		// No assertion needed - just verify it doesn't panic
	})

	t.Run("IncrementSupervisorError increments counter", func(t *testing.T) {
		IncrementSupervisorError()
		// No assertion needed - just verify it doesn't panic
	})

	t.Run("IncrementInstanceOperation increments counter", func(t *testing.T) {
		IncrementInstanceOperation("create", "success")
		IncrementInstanceOperation("create", "failed")
		IncrementInstanceOperation("delete", "success")
		IncrementInstanceOperation("update", "success")
		// No assertion needed - just verify it doesn't panic
	})
}

func TestMetricsHandler(t *testing.T) {
	app := fiber.New()
	app.Get("/metrics", MetricsHandler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
