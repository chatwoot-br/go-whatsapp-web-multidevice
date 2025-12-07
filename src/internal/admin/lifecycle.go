package admin

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/abrander/go-supervisord"
	"github.com/sirupsen/logrus"
)

// InstanceState represents the state of a GOWA instance
type InstanceState string

const (
	StateRunning  InstanceState = "RUNNING"
	StateStopped  InstanceState = "STOPPED"
	StateStarting InstanceState = "STARTING"
	StateFatal    InstanceState = "FATAL"
	StateUnknown  InstanceState = "UNKNOWN"
)

// Instance represents a GOWA instance managed by supervisord
type Instance struct {
	Port     int           `json:"port"`
	State    InstanceState `json:"state"`
	PID      int           `json:"pid"`
	Uptime   time.Duration `json:"uptime"`
	LogFiles LogFiles      `json:"logs"`
}

// LogFiles contains the paths to log files for an instance
type LogFiles struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

// ILifecycleManager defines the interface for lifecycle management
type ILifecycleManager interface {
	CreateInstance(ctx context.Context, port int) (*Instance, error)
	CreateInstanceWithConfig(ctx context.Context, port int, customConfig *InstanceConfig) (*Instance, error)
	UpdateInstanceConfig(ctx context.Context, port int, customConfig *InstanceConfig) (*Instance, error)
	ListInstances(ctx context.Context) ([]*Instance, error)
	GetInstance(ctx context.Context, port int) (*Instance, error)
	DeleteInstance(ctx context.Context, port int) error
	IsHealthy() bool
	Ping() error
}

// LifecycleManager handles creation and deletion of GOWA instances
type LifecycleManager struct {
	supervisor   *SupervisorClient
	configWriter *ConfigWriter
	lockManager  *LockManager
	logger       *logrus.Logger
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(supervisor *SupervisorClient, configWriter *ConfigWriter, logger *logrus.Logger) *LifecycleManager {
	return &LifecycleManager{
		supervisor:   supervisor,
		configWriter: configWriter,
		lockManager:  NewLockManager(),
		logger:       logger,
	}
}

// CreateInstance creates a new GOWA instance on the specified port
func (lm *LifecycleManager) CreateInstance(ctx context.Context, port int) (*Instance, error) {
	return lm.CreateInstanceWithConfig(ctx, port, nil)
}

// CreateInstanceWithConfig creates a new GOWA instance with custom configuration
func (lm *LifecycleManager) CreateInstanceWithConfig(ctx context.Context, port int, customConfig *InstanceConfig) (*Instance, error) {
	// Validate port
	if err := lm.validatePort(port); err != nil {
		return nil, fmt.Errorf("port validation failed: %w", err)
	}

	// Create lock context with timeout
	lockCtx, lockCancel := context.WithTimeout(ctx, DefaultLockTimeout)
	defer lockCancel()

	// Acquire lock for this port with context
	lockFile, err := lm.lockManager.AcquireLockWithContext(lockCtx, port)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lm.lockManager.ReleaseLock(lockFile)

	programName := fmt.Sprintf("gowa_%d", port)

	// Check if instance already exists
	if lm.instanceExists(programName) {
		return nil, fmt.Errorf("instance on port %d already exists", port)
	}

	// Check if port is available
	if !lm.isPortAvailable(port) {
		return nil, fmt.Errorf("port %d is already in use", port)
	}

	lm.logger.Infof("Creating instance on port %d", port)

	// Write configuration file with custom config if provided
	if err := lm.writeConfigForInstance(port, customConfig); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	// Use Update() to reload configuration and add process group
	client := lm.supervisor.GetClient()
	if err := client.Update(); err != nil {
		// Clean up config file on failure
		lm.configWriter.RemoveConfig(port)
		return nil, fmt.Errorf("failed to update supervisord configuration: %w", err)
	}

	// Try to start the process (it might already be started due to autostart=true)
	if err := client.StartProcess(programName, true); err != nil {
		// Check if the error is because it's already started
		if strings.Contains(err.Error(), "ALREADY_STARTED") {
			// This is expected when autostart=true, just log it
			lm.logger.Debugf("Process %s was already started by supervisord autostart", programName)
		} else {
			// Try to remove process group and config on failure
			lm.cleanupFailedInstance(programName, port)
			return nil, fmt.Errorf("failed to start process %s: %w", programName, err)
		}
	}

	// Wait for the process to be running with timeout
	instance, err := lm.waitForInstanceState(ctx, port, StateRunning, 30*time.Second)
	if err != nil {
		// Clean up if instance didn't start properly
		lm.cleanupFailedInstance(programName, port)
		return nil, fmt.Errorf("instance failed to start within timeout: %w", err)
	}

	lm.logger.Infof("Successfully created instance on port %d", port)
	return instance, nil
}

// DeleteInstance deletes a GOWA instance on the specified port
func (lm *LifecycleManager) DeleteInstance(ctx context.Context, port int) error {
	// Create lock context with timeout
	lockCtx, lockCancel := context.WithTimeout(ctx, DefaultLockTimeout)
	defer lockCancel()

	// Acquire lock for this port with context
	lockFile, err := lm.lockManager.AcquireLockWithContext(lockCtx, port)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lm.lockManager.ReleaseLock(lockFile)

	programName := fmt.Sprintf("gowa_%d", port)

	lm.logger.Infof("Deleting instance on port %d", port)

	client := lm.supervisor.GetClient()

	// Stop the process if it's running
	if lm.instanceExists(programName) {
		if err := client.StopProcess(programName, true); err != nil {
			lm.logger.Warnf("Failed to stop process %s: %v", programName, err)
		}

		// Remove process group
		if err := client.RemoveProcessGroup(programName); err != nil {
			lm.logger.Warnf("Failed to remove process group %s: %v", programName, err)
		}
	}

	// Remove storage directory (database, chat storage, media, etc.)
	if err := lm.configWriter.RemoveStorage(port); err != nil {
		lm.logger.Warnf("Failed to remove storage for port %d: %v", port, err)
		// Continue with cleanup - storage removal is best-effort
	}

	// Remove log files
	if err := lm.configWriter.RemoveLogs(port); err != nil {
		lm.logger.Warnf("Failed to remove logs for port %d: %v", port, err)
		// Continue with cleanup - log removal is best-effort
	}

	// Remove configuration file
	if err := lm.configWriter.RemoveConfig(port); err != nil {
		return fmt.Errorf("failed to remove config: %w", err)
	}

	lm.logger.Infof("Successfully deleted instance on port %d", port)
	return nil
}

// UpdateInstanceConfig updates a GOWA instance configuration with new settings
func (lm *LifecycleManager) UpdateInstanceConfig(ctx context.Context, port int, customConfig *InstanceConfig) (*Instance, error) {
	// Create lock context with timeout
	lockCtx, lockCancel := context.WithTimeout(ctx, DefaultLockTimeout)
	defer lockCancel()

	// Acquire lock for this port with context
	lockFile, err := lm.lockManager.AcquireLockWithContext(lockCtx, port)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lm.lockManager.ReleaseLock(lockFile)

	programName := fmt.Sprintf("gowa_%d", port)

	lm.logger.Infof("Updating instance configuration on port %d", port)

	// Check if instance exists
	if !lm.instanceExists(programName) {
		return nil, fmt.Errorf("instance on port %d not found", port)
	}

	client := lm.supervisor.GetClient()

	// Stop the process if it's running
	lm.logger.Infof("Stopping instance on port %d for configuration update", port)
	if err := client.StopProcess(programName, true); err != nil {
		lm.logger.Warnf("Failed to stop process %s: %v", programName, err)
	}

	// Remove the current process group
	if err := client.RemoveProcessGroup(programName); err != nil {
		lm.logger.Warnf("Failed to remove process group %s: %v", programName, err)
	}

	// Set the port in the custom config
	if customConfig == nil {
		customConfig = DefaultInstanceConfig()
	}
	customConfig.Port = port

	// Write the new configuration
	if err := lm.configWriter.WriteConfigWithCustom(port, customConfig); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	// Add the new configuration to supervisord
	if err := client.Update(); err != nil {
		return nil, fmt.Errorf("failed to update supervisord configuration: %w", err)
	}

	// Wait a moment for supervisord to process the reload
	time.Sleep(1 * time.Second)

	// Start the process with new configuration
	lm.logger.Infof("Starting instance on port %d with updated configuration", port)
	if err := client.StartProcess(programName, true); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Wait for the instance to be in running state
	instance, err := lm.waitForInstanceState(ctx, port, StateRunning, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("instance failed to start: %w", err)
	}

	lm.logger.Infof("Successfully updated instance configuration on port %d", port)
	return instance, nil
}

// ListInstances returns a list of all GOWA instances
// Note: This is a read operation and does not acquire locks for consistency
// Callers may observe intermediate state during concurrent write operations
func (lm *LifecycleManager) ListInstances(ctx context.Context) ([]*Instance, error) {
	client := lm.supervisor.GetClient()

	processInfos, err := client.GetAllProcessInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get process info: %w", err)
	}

	var instances []*Instance
	for _, info := range processInfos {
		if strings.HasPrefix(info.Name, "gowa_") {
			instance := lm.processInfoToInstance(&info)
			if instance != nil {
				instances = append(instances, instance)
			}
		}
	}

	return instances, nil
}

// GetInstance returns information about a specific instance
// Note: This is a read operation and does not acquire locks for consistency
// Callers may observe intermediate state during concurrent write operations
func (lm *LifecycleManager) GetInstance(ctx context.Context, port int) (*Instance, error) {
	programName := fmt.Sprintf("gowa_%d", port)

	client := lm.supervisor.GetClient()
	info, err := client.GetProcessInfo(programName)
	if err != nil {
		return nil, fmt.Errorf("instance on port %d not found: %w", port, err)
	}

	return lm.processInfoToInstance(info), nil
}

// validatePort validates that the port is in a valid range
func (lm *LifecycleManager) validatePort(port int) error {
	if port < 1024 || port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535, got %d", port)
	}
	return nil
}

// isPortAvailable checks if a port is available for binding
func (lm *LifecycleManager) isPortAvailable(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// instanceExists checks if a supervisord process with the given name exists
func (lm *LifecycleManager) instanceExists(programName string) bool {
	client := lm.supervisor.GetClient()
	_, err := client.GetProcessInfo(programName)
	return err == nil
}

// processInfoToInstance converts supervisord process info to Instance
func (lm *LifecycleManager) processInfoToInstance(info *supervisord.ProcessInfo) *Instance {
	// Extract port from program name (gowa_3001 -> 3001)
	portStr := strings.TrimPrefix(info.Name, "gowa_")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil
	}

	// Map supervisord state to our InstanceState
	var state InstanceState
	switch info.StateName {
	case "RUNNING":
		state = StateRunning
	case "STOPPED":
		state = StateStopped
	case "STARTING":
		state = StateStarting
	case "FATAL":
		state = StateFatal
	default:
		state = StateUnknown
	}

	// Calculate uptime
	var uptime time.Duration
	if info.Start > 0 {
		uptime = time.Since(time.Unix(int64(info.Start), 0))
	}

	// Get log file paths
	logFiles := LogFiles{
		Stdout: info.StdoutLogfile,
		Stderr: info.StderrLogfile,
	}

	return &Instance{
		Port:     port,
		State:    state,
		PID:      info.Pid,
		Uptime:   uptime,
		LogFiles: logFiles,
	}
}

// waitForInstanceState waits for an instance to reach a specific state
func (lm *LifecycleManager) waitForInstanceState(ctx context.Context, port int, targetState InstanceState, timeout time.Duration) (*Instance, error) {
	programName := fmt.Sprintf("gowa_%d", port)
	client := lm.supervisor.GetClient()

	// Create timeout context if parent doesn't have a shorter deadline
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for instance to reach state %s: %w", targetState, ctx.Err())
		case <-ticker.C:
			info, err := client.GetProcessInfo(programName)
			if err != nil {
				continue
			}

			instance := lm.processInfoToInstance(info)
			if instance != nil && instance.State == targetState {
				return instance, nil
			}

			if instance != nil && instance.State == StateFatal {
				return nil, fmt.Errorf("instance entered FATAL state")
			}
		}
	}
}

// cleanupFailedInstance cleans up a failed instance creation
func (lm *LifecycleManager) cleanupFailedInstance(programName string, port int) {
	client := lm.supervisor.GetClient()

	// Try to stop and remove the process group
	client.StopProcess(programName, false)
	client.RemoveProcessGroup(programName)

	// Remove configuration file
	lm.configWriter.RemoveConfig(port)
}

// writeConfigForInstance writes configuration for an instance with optional custom config
func (lm *LifecycleManager) writeConfigForInstance(port int, customConfig *InstanceConfig) error {
	if customConfig == nil {
		// Use default configuration
		return lm.configWriter.WriteConfig(port)
	}

	// Create a custom config writer with the provided configuration
	customConfig.Port = port
	customWriter, err := NewConfigWriter(customConfig)
	if err != nil {
		return fmt.Errorf("failed to create custom config writer: %w", err)
	}

	return customWriter.WriteConfig(port)
}

// IsHealthy checks if the supervisor is healthy
func (lm *LifecycleManager) IsHealthy() bool {
	return lm.supervisor.IsHealthy()
}

// Ping checks if the supervisor is reachable
func (lm *LifecycleManager) Ping() error {
	return lm.supervisor.Ping()
}
