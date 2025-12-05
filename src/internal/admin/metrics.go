package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// instancesRunning tracks the number of GOWA instances currently running
	instancesRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gowa_instances_running",
		Help: "Number of GOWA instances currently running",
	})

	// apiRequestsTotal tracks total API requests by method, path and status
	apiRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gowa_admin_api_requests_total",
		Help: "Total API requests by method, path and status",
	}, []string{"method", "path", "status"})

	// supervisorErrorsTotal tracks total supervisord errors
	supervisorErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gowa_supervisor_errors_total",
		Help: "Total supervisord errors",
	})

	// instanceOperationsTotal tracks instance create/delete operations
	instanceOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gowa_instance_operations_total",
		Help: "Total instance operations by type and result",
	}, []string{"operation", "result"})
)

// SetInstancesRunning sets the gauge for running instances
func SetInstancesRunning(count int) {
	instancesRunning.Set(float64(count))
}

// IncrementAPIRequest increments the API request counter
func IncrementAPIRequest(method, path, status string) {
	apiRequestsTotal.WithLabelValues(method, path, status).Inc()
}

// IncrementSupervisorError increments the supervisor error counter
func IncrementSupervisorError() {
	supervisorErrorsTotal.Inc()
}

// IncrementInstanceOperation increments the instance operation counter
func IncrementInstanceOperation(operation, result string) {
	instanceOperationsTotal.WithLabelValues(operation, result).Inc()
}

// MetricsHandler returns the Prometheus metrics handler for Fiber
func MetricsHandler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}
