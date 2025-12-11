package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AdminAPI handles HTTP requests for instance management
type AdminAPI struct {
	lifecycle   ILifecycleManager
	logger      *logrus.Logger
	auditLogger *AuditLogger
	adminToken  string
}

// CreateInstanceRequest represents the request body for creating an instance
type CreateInstanceRequest struct {
	Port              int    `json:"port" validate:"required,min=1024,max=65535"`
	BasicAuth         string `json:"basic_auth,omitempty"`
	Debug             *bool  `json:"debug,omitempty"`
	OS                string `json:"os,omitempty"`
	AccountValidation *bool  `json:"account_validation,omitempty"`
	BasePath          string `json:"base_path,omitempty"`
	AutoReply         string `json:"auto_reply,omitempty"`
	AutoMarkRead      *bool  `json:"auto_mark_read,omitempty"`
	Webhook           string `json:"webhook,omitempty"`
	WebhookSecret     string `json:"webhook_secret,omitempty"`
	ChatStorage       *bool  `json:"chat_storage,omitempty"`
}

// UpdateInstanceRequest represents the request body for updating an instance
type UpdateInstanceRequest struct {
	BasicAuth         *string `json:"basic_auth,omitempty"`
	Debug             *bool   `json:"debug,omitempty"`
	OS                *string `json:"os,omitempty"`
	AccountValidation *bool   `json:"account_validation,omitempty"`
	BasePath          *string `json:"base_path,omitempty"`
	AutoReply         *string `json:"auto_reply,omitempty"`
	AutoMarkRead      *bool   `json:"auto_mark_read,omitempty"`
	Webhook           *string `json:"webhook,omitempty"`
	WebhookSecret     *string `json:"webhook_secret,omitempty"`
	ChatStorage       *bool   `json:"chat_storage,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Timestamp string `json:"timestamp"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id"`
	Timestamp string      `json:"timestamp"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
	Supervisor bool      `json:"supervisor_healthy"`
	Version    string    `json:"version"`
}

// CleanupRequest represents the request body for cleanup operation
type CleanupRequest struct {
	RetentionDays int      `json:"retention_days,omitempty"`
	Directories   []string `json:"directories,omitempty"`
	DryRun        bool     `json:"dry_run,omitempty"`
}

// CleanupResult represents the result of cleaning a single directory
type CleanupResult struct {
	Directory    string   `json:"directory"`
	FilesDeleted int      `json:"files_deleted"`
	DirsDeleted  int      `json:"dirs_deleted"`
	BytesFreed   int64    `json:"bytes_freed"`
	Errors       []string `json:"errors,omitempty"`
}

// CleanupResponse represents the response from cleanup operation
type CleanupResponse struct {
	RetentionDays int             `json:"retention_days"`
	DryRun        bool            `json:"dry_run"`
	Results       []CleanupResult `json:"results"`
	TotalFiles    int             `json:"total_files_deleted"`
	TotalDirs     int             `json:"total_dirs_deleted"`
	TotalBytes    int64           `json:"total_bytes_freed"`
}

// NewAdminAPI creates a new AdminAPI instance
func NewAdminAPI(lifecycle ILifecycleManager, logger *logrus.Logger) (*AdminAPI, error) {
	adminToken := os.Getenv("ADMIN_TOKEN")
	if adminToken == "" {
		return nil, fmt.Errorf("ADMIN_TOKEN environment variable is required")
	}

	return &AdminAPI{
		lifecycle:   lifecycle,
		logger:      logger,
		auditLogger: NewAuditLogger(logger),
		adminToken:  adminToken,
	}, nil
}

// SetupRoutes configures the Fiber app with admin routes
func (api *AdminAPI) SetupRoutes(app *fiber.App) {
	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} ${status} - ${method} ${path} ${latency}\n",
	}))
	app.Use(cors.New())
	app.Use(api.requestIDMiddleware)
	app.Use(api.timeoutMiddleware)
	app.Use(api.metricsMiddleware)

	// Health endpoints
	app.Get("/healthz", api.healthHandler)
	app.Get("/readyz", api.readinessHandler)

	// Metrics endpoint (Prometheus)
	app.Get("/metrics", MetricsHandler())

	// Admin routes with authentication
	admin := app.Group("/admin", api.authMiddleware)
	admin.Post("/instances", api.createInstanceHandler)
	admin.Get("/instances", api.listInstancesHandler)
	admin.Get("/instances/:port", api.getInstanceHandler)
	admin.Patch("/instances/:port", api.updateInstanceHandler)
	admin.Delete("/instances/:port", api.deleteInstanceHandler)
	admin.Post("/cleanup", api.cleanupHandler)
}

// timeoutMiddleware adds a timeout context to each request
func (api *AdminAPI) timeoutMiddleware(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 60*time.Second)
	defer cancel()
	c.SetUserContext(ctx)
	return c.Next()
}

// metricsMiddleware tracks API request metrics
func (api *AdminAPI) metricsMiddleware(c *fiber.Ctx) error {
	err := c.Next()
	// Record metrics after request completes
	IncrementAPIRequest(c.Method(), c.Path(), strconv.Itoa(c.Response().StatusCode()))
	return err
}

// requestIDMiddleware adds a request ID to each request
func (api *AdminAPI) requestIDMiddleware(c *fiber.Ctx) error {
	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	c.Locals("request_id", requestID)
	c.Set("X-Request-ID", requestID)
	return c.Next()
}

// authMiddleware validates bearer token authentication
func (api *AdminAPI) authMiddleware(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	if auth == "" {
		return api.errorResponse(c, http.StatusUnauthorized, "missing_authorization", "Authorization header is required")
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		return api.errorResponse(c, http.StatusUnauthorized, "invalid_authorization", "Authorization must use Bearer token")
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	if token != api.adminToken {
		return api.errorResponse(c, http.StatusUnauthorized, "invalid_token", "Invalid or expired token")
	}

	return c.Next()
}

// createInstanceHandler handles POST /admin/instances
func (api *AdminAPI) createInstanceHandler(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := api.getRequestID(c)

	var req CreateInstanceRequest
	if err := c.BodyParser(&req); err != nil {
		return api.errorResponse(c, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
	}

	// Validate port range
	if req.Port < 1024 || req.Port > 65535 {
		return api.errorResponse(c, http.StatusBadRequest, "invalid_port", "Port must be between 1024 and 65535")
	}

	api.logger.Infof("Creating instance on port %d with custom config", req.Port)

	// Create custom configuration if any fields are provided
	var customConfig *InstanceConfig
	if api.hasCustomConfig(req) {
		customConfig = api.buildInstanceConfig(req)
	}

	// Get context from Fiber request with timeout
	ctx := api.getRequestContext(c)

	var instance *Instance
	var err error

	if customConfig != nil {
		instance, err = api.lifecycle.CreateInstanceWithConfig(ctx, req.Port, customConfig)
	} else {
		instance, err = api.lifecycle.CreateInstance(ctx, req.Port)
	}
	if err != nil {
		api.logger.Errorf("Failed to create instance on port %d: %v", req.Port, err)
		IncrementInstanceOperation("create", "failed")
		api.auditLogger.LogCreate(req.Port, requestID, "failed", err, time.Since(startTime))

		if strings.Contains(err.Error(), "already exists") {
			return api.errorResponse(c, http.StatusConflict, "instance_exists", err.Error())
		}

		if strings.Contains(err.Error(), "port") && strings.Contains(err.Error(), "in use") {
			return api.errorResponse(c, http.StatusConflict, "port_in_use", err.Error())
		}

		if strings.Contains(err.Error(), "locked") || strings.Contains(err.Error(), "cancelled") {
			return api.errorResponse(c, http.StatusConflict, "port_locked", err.Error())
		}

		IncrementSupervisorError()
		status, errorCode := api.classifySupervisorError(err)
		return api.errorResponse(c, status, errorCode, err.Error())
	}

	IncrementInstanceOperation("create", "success")
	api.auditLogger.LogCreate(req.Port, requestID, "success", nil, time.Since(startTime))
	return api.successResponse(c, http.StatusCreated, instance, "Instance created successfully")
}

// listInstancesHandler handles GET /admin/instances
func (api *AdminAPI) listInstancesHandler(c *fiber.Ctx) error {
	ctx := api.getRequestContext(c)

	instances, err := api.lifecycle.ListInstances(ctx)
	if err != nil {
		api.logger.Errorf("Failed to list instances: %v", err)
		IncrementSupervisorError()
		status, errorCode := api.classifySupervisorError(err)
		return api.errorResponse(c, status, errorCode, "Failed to retrieve instances")
	}

	// Update running instances gauge
	runningCount := 0
	for _, inst := range instances {
		if inst.State == StateRunning {
			runningCount++
		}
	}
	SetInstancesRunning(runningCount)

	return api.successResponse(c, http.StatusOK, instances, "Instances retrieved successfully")
}

// getInstanceHandler handles GET /admin/instances/:port
func (api *AdminAPI) getInstanceHandler(c *fiber.Ctx) error {
	portStr := c.Params("port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return api.errorResponse(c, http.StatusBadRequest, "invalid_port", "Port must be a valid integer")
	}

	ctx := api.getRequestContext(c)

	instance, err := api.lifecycle.GetInstance(ctx, port)
	if err != nil {
		api.logger.Errorf("Failed to get instance on port %d: %v", port, err)

		if strings.Contains(err.Error(), "not found") {
			return api.errorResponse(c, http.StatusNotFound, "instance_not_found", fmt.Sprintf("Instance on port %d not found", port))
		}

		status, errorCode := api.classifySupervisorError(err)
		return api.errorResponse(c, status, errorCode, "Failed to retrieve instance")
	}

	return api.successResponse(c, http.StatusOK, instance, "Instance retrieved successfully")
}

// deleteInstanceHandler handles DELETE /admin/instances/:port
func (api *AdminAPI) deleteInstanceHandler(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := api.getRequestID(c)

	portStr := c.Params("port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return api.errorResponse(c, http.StatusBadRequest, "invalid_port", "Port must be a valid integer")
	}

	api.logger.Infof("Deleting instance on port %d", port)

	ctx := api.getRequestContext(c)

	err = api.lifecycle.DeleteInstance(ctx, port)
	if err != nil {
		api.logger.Errorf("Failed to delete instance on port %d: %v", port, err)
		IncrementInstanceOperation("delete", "failed")
		api.auditLogger.LogDelete(port, requestID, "failed", err, time.Since(startTime))

		if strings.Contains(err.Error(), "not found") {
			return api.errorResponse(c, http.StatusNotFound, "instance_not_found", fmt.Sprintf("Instance on port %d not found", port))
		}

		if strings.Contains(err.Error(), "locked") || strings.Contains(err.Error(), "cancelled") {
			return api.errorResponse(c, http.StatusConflict, "port_locked", err.Error())
		}

		IncrementSupervisorError()
		status, errorCode := api.classifySupervisorError(err)
		return api.errorResponse(c, status, errorCode, "Failed to delete instance")
	}

	IncrementInstanceOperation("delete", "success")
	api.auditLogger.LogDelete(port, requestID, "success", nil, time.Since(startTime))
	return api.successResponse(c, http.StatusOK, nil, "Instance deleted successfully")
}

// updateInstanceHandler handles PATCH /admin/instances/:port
func (api *AdminAPI) updateInstanceHandler(c *fiber.Ctx) error {
	startTime := time.Now()
	requestID := api.getRequestID(c)

	portStr := c.Params("port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return api.errorResponse(c, http.StatusBadRequest, "invalid_port", "Port must be a valid integer")
	}

	var req UpdateInstanceRequest
	if err := c.BodyParser(&req); err != nil {
		return api.errorResponse(c, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
	}

	api.logger.Infof("Updating instance on port %d with new configuration", port)

	ctx := api.getRequestContext(c)

	// Build the custom configuration from the update request
	customConfig := api.buildInstanceConfigFromUpdate(port, req)

	instance, err := api.lifecycle.UpdateInstanceConfig(ctx, port, customConfig)
	if err != nil {
		api.logger.Errorf("Failed to update instance on port %d: %v", port, err)
		IncrementInstanceOperation("update", "failed")
		api.auditLogger.LogUpdate(port, requestID, "failed", err, time.Since(startTime))

		if strings.Contains(err.Error(), "not found") {
			return api.errorResponse(c, http.StatusNotFound, "instance_not_found", err.Error())
		}

		if strings.Contains(err.Error(), "locked") || strings.Contains(err.Error(), "cancelled") {
			return api.errorResponse(c, http.StatusConflict, "port_locked", err.Error())
		}

		IncrementSupervisorError()
		status, errorCode := api.classifySupervisorError(err)
		return api.errorResponse(c, status, errorCode, err.Error())
	}

	IncrementInstanceOperation("update", "success")
	api.auditLogger.LogUpdate(port, requestID, "success", nil, time.Since(startTime))
	return api.successResponse(c, http.StatusOK, instance, "Instance configuration updated successfully")
}

// healthHandler handles GET /healthz
func (api *AdminAPI) healthHandler(c *fiber.Ctx) error {
	response := HealthResponse{
		Status:     "healthy",
		Timestamp:  time.Now(),
		Supervisor: api.lifecycle.IsHealthy(),
		Version:    config.AppVersion,
	}

	status := http.StatusOK
	if !response.Supervisor {
		response.Status = "degraded"
		status = http.StatusServiceUnavailable
	}

	return c.Status(status).JSON(response)
}

// readinessHandler handles GET /readyz
func (api *AdminAPI) readinessHandler(c *fiber.Ctx) error {
	// Check if supervisord is reachable
	if err := api.lifecycle.Ping(); err != nil {
		return api.errorResponse(c, http.StatusServiceUnavailable, "supervisor_unreachable", "Supervisord is not reachable")
	}

	return api.successResponse(c, http.StatusOK, nil, "Service is ready")
}

// cleanupHandler handles POST /admin/cleanup
// Cleans up old files from specified directories
func (api *AdminAPI) cleanupHandler(c *fiber.Ctx) error {
	requestID := api.getRequestID(c)

	var req CleanupRequest
	if err := c.BodyParser(&req); err != nil && err.Error() != "Unprocessable Entity" {
		return api.errorResponse(c, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
	}

	// Default retention days from env or fallback to 7
	retentionDays := req.RetentionDays
	if retentionDays <= 0 {
		envDays := os.Getenv("CLEANUP_RETENTION_DAYS")
		if envDays != "" {
			if days, err := strconv.Atoi(envDays); err == nil && days > 0 {
				retentionDays = days
			}
		}
		if retentionDays <= 0 {
			retentionDays = 7
		}
	}

	// Default directories from env or fallback to standard paths
	directories := req.Directories
	if len(directories) == 0 {
		envDirs := os.Getenv("CLEANUP_DIRECTORIES")
		if envDirs != "" {
			directories = strings.Split(envDirs, ",")
		} else {
			// Default directories for media cleanup
			directories = []string{
				"/app/statics/media",
				"/app/statics/qrcode",
				"/app/statics/senditems",
			}
		}
	}

	api.logger.WithFields(logrus.Fields{
		"request_id":     requestID,
		"retention_days": retentionDays,
		"directories":    directories,
		"dry_run":        req.DryRun,
	}).Info("Starting cleanup operation")

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	response := CleanupResponse{
		RetentionDays: retentionDays,
		DryRun:        req.DryRun,
		Results:       make([]CleanupResult, 0, len(directories)),
	}

	for _, dir := range directories {
		result := api.cleanupDirectory(dir, cutoffTime, req.DryRun)
		response.Results = append(response.Results, result)
		response.TotalFiles += result.FilesDeleted
		response.TotalDirs += result.DirsDeleted
		response.TotalBytes += result.BytesFreed
	}

	api.logger.WithFields(logrus.Fields{
		"request_id":    requestID,
		"files_deleted": response.TotalFiles,
		"dirs_deleted":  response.TotalDirs,
		"bytes_freed":   response.TotalBytes,
		"dry_run":       req.DryRun,
	}).Info("Cleanup operation completed")

	return api.successResponse(c, http.StatusOK, response, "Cleanup completed successfully")
}

// cleanupDirectory cleans files older than cutoffTime from a directory
func (api *AdminAPI) cleanupDirectory(dir string, cutoffTime time.Time, dryRun bool) CleanupResult {
	result := CleanupResult{
		Directory: dir,
		Errors:    make([]string, 0),
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		api.logger.Debugf("Directory does not exist, skipping: %s", dir)
		return result
	}

	// First pass: delete old files
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("error accessing %s: %v", path, err))
			return nil // Continue walking despite errors
		}

		// Skip the root directory itself
		if path == dir {
			return nil
		}

		// Only process files (not directories) in first pass
		if !info.IsDir() && info.ModTime().Before(cutoffTime) {
			if dryRun {
				result.FilesDeleted++
				result.BytesFreed += info.Size()
				api.logger.Debugf("Would delete file: %s (size: %d, mtime: %s)", path, info.Size(), info.ModTime())
			} else {
				if err := os.Remove(path); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("failed to delete %s: %v", path, err))
				} else {
					result.FilesDeleted++
					result.BytesFreed += info.Size()
					api.logger.Debugf("Deleted file: %s", path)
				}
			}
		}
		return nil
	})

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("walk error: %v", err))
	}

	// Second pass: delete empty directories (bottom-up)
	api.cleanupEmptyDirs(dir, dryRun, &result)

	return result
}

// cleanupEmptyDirs removes empty directories recursively (bottom-up)
func (api *AdminAPI) cleanupEmptyDirs(dir string, dryRun bool, result *CleanupResult) {
	// Collect all subdirectories first
	var subdirs []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && path != dir {
			subdirs = append(subdirs, path)
		}
		return nil
	})

	if err != nil {
		return
	}

	// Process directories in reverse order (deepest first)
	for i := len(subdirs) - 1; i >= 0; i-- {
		subdir := subdirs[i]
		entries, err := os.ReadDir(subdir)
		if err != nil {
			continue
		}

		if len(entries) == 0 {
			if dryRun {
				result.DirsDeleted++
				api.logger.Debugf("Would delete empty directory: %s", subdir)
			} else {
				if err := os.Remove(subdir); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("failed to delete dir %s: %v", subdir, err))
				} else {
					result.DirsDeleted++
					api.logger.Debugf("Deleted empty directory: %s", subdir)
				}
			}
		}
	}
}

// errorResponse sends a standardized error response
func (api *AdminAPI) errorResponse(c *fiber.Ctx, status int, errorCode, message string) error {
	requestID := api.getRequestID(c)

	response := ErrorResponse{
		Error:     errorCode,
		Message:   message,
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Log error for debugging
	api.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"status":     status,
		"error_code": errorCode,
		"message":    message,
	}).Error("API error response")

	return c.Status(status).JSON(response)
}

// successResponse sends a standardized success response
func (api *AdminAPI) successResponse(c *fiber.Ctx, status int, data interface{}, message string) error {
	requestID := api.getRequestID(c)

	response := SuccessResponse{
		Data:      data,
		Message:   message,
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	return c.Status(status).JSON(response)
}

// getRequestID retrieves the request ID from context
func (api *AdminAPI) getRequestID(c *fiber.Ctx) string {
	if requestID, ok := c.Locals("request_id").(string); ok {
		return requestID
	}
	return "unknown"
}

// getRequestContext returns the request context (timeout is set by middleware)
func (api *AdminAPI) getRequestContext(c *fiber.Ctx) context.Context {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

// classifySupervisorError maps supervisor errors to appropriate HTTP status codes
func (api *AdminAPI) classifySupervisorError(err error) (int, string) {
	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "failed to ping") {
		return http.StatusBadGateway, "supervisor_unreachable"
	}
	if strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "deadline exceeded") ||
		strings.Contains(errMsg, "context deadline") {
		return http.StatusGatewayTimeout, "supervisor_timeout"
	}
	return http.StatusInternalServerError, "supervisor_error"
}

// StartServer starts the admin HTTP server
func (api *AdminAPI) StartServer(port string) error {
	app := fiber.New(fiber.Config{
		ErrorHandler: api.errorHandler,
		JSONEncoder:  json.Marshal,
		JSONDecoder:  json.Unmarshal,
	})

	api.SetupRoutes(app)

	api.logger.Infof("Starting admin server on port %s", port)
	return app.Listen(":" + port)
}

// errorHandler handles uncaught errors
func (api *AdminAPI) errorHandler(c *fiber.Ctx, err error) error {
	// Default to 500 Internal Server Error
	code := http.StatusInternalServerError
	message := "Internal Server Error"

	// Handle Fiber errors
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	api.logger.WithFields(logrus.Fields{
		"path":   c.Path(),
		"method": c.Method(),
		"error":  err.Error(),
	}).Error("Unhandled error")

	return api.errorResponse(c, code, "internal_error", message)
}

// hasCustomConfig checks if the request contains any custom configuration
func (api *AdminAPI) hasCustomConfig(req CreateInstanceRequest) bool {
	return req.BasicAuth != "" ||
		req.Debug != nil ||
		req.OS != "" ||
		req.AccountValidation != nil ||
		req.BasePath != "" ||
		req.AutoReply != "" ||
		req.AutoMarkRead != nil ||
		req.Webhook != "" ||
		req.WebhookSecret != "" ||
		req.ChatStorage != nil
}

// buildInstanceConfig creates an InstanceConfig from the API request
func (api *AdminAPI) buildInstanceConfig(req CreateInstanceRequest) *InstanceConfig {
	// Start with default configuration
	config := DefaultInstanceConfig()

	// Override with values from request
	if req.BasicAuth != "" {
		config.BasicAuth = req.BasicAuth
	}
	if req.Debug != nil {
		config.Debug = *req.Debug
	}
	if req.OS != "" {
		config.OS = req.OS
	}
	if req.AccountValidation != nil {
		config.AccountValidation = *req.AccountValidation
	}
	if req.BasePath != "" {
		config.BasePath = req.BasePath
	}
	if req.AutoReply != "" {
		config.AutoReply = req.AutoReply
	}
	if req.AutoMarkRead != nil {
		config.AutoMarkRead = *req.AutoMarkRead
	}
	if req.Webhook != "" {
		config.Webhook = req.Webhook
	}
	if req.WebhookSecret != "" {
		config.WebhookSecret = req.WebhookSecret
	}
	if req.ChatStorage != nil {
		config.ChatStorage = *req.ChatStorage
	}

	return config
}

// buildInstanceConfigFromUpdate creates an InstanceConfig from the API update request
func (api *AdminAPI) buildInstanceConfigFromUpdate(port int, req UpdateInstanceRequest) *InstanceConfig {
	// Start with default configuration
	config := DefaultInstanceConfig()
	config.Port = port

	// Override with values from request only if they are provided
	if req.BasicAuth != nil {
		config.BasicAuth = *req.BasicAuth
	}
	if req.Debug != nil {
		config.Debug = *req.Debug
	}
	if req.OS != nil {
		config.OS = *req.OS
	}
	if req.AccountValidation != nil {
		config.AccountValidation = *req.AccountValidation
	}
	if req.BasePath != nil {
		config.BasePath = *req.BasePath
	}
	if req.AutoReply != nil {
		config.AutoReply = *req.AutoReply
	}
	if req.AutoMarkRead != nil {
		config.AutoMarkRead = *req.AutoMarkRead
	}
	if req.Webhook != nil {
		config.Webhook = *req.Webhook
	}
	if req.WebhookSecret != nil {
		config.WebhookSecret = *req.WebhookSecret
	}
	if req.ChatStorage != nil {
		config.ChatStorage = *req.ChatStorage
	}

	return config
}
