package admin

import (
	"testing"
	"time"

	"github.com/abrander/go-supervisord"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewLifecycleManager(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("creates lifecycle manager with dependencies", func(t *testing.T) {
		config := DefaultInstanceConfig()
		configWriter, err := NewConfigWriter(config)
		assert.NoError(t, err)

		// Note: We can't test with a real supervisor client in unit tests
		// This test just verifies the constructor works
		lm := NewLifecycleManager(nil, configWriter, logger)
		assert.NotNil(t, lm)
		assert.NotNil(t, lm.configWriter)
		assert.NotNil(t, lm.lockManager)
		assert.NotNil(t, lm.logger)
	})
}

func TestLifecycleManager_validatePort(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := DefaultInstanceConfig()
	configWriter, _ := NewConfigWriter(config)
	lm := NewLifecycleManager(nil, configWriter, logger)

	tests := []struct {
		name        string
		port        int
		expectError bool
	}{
		{"valid port 3001", 3001, false},
		{"valid port 8080", 8080, false},
		{"valid port 65535", 65535, false},
		{"valid port 1024", 1024, false},
		{"invalid port 0", 0, true},
		{"invalid port negative", -1, true},
		{"invalid port too low", 1023, true},
		{"invalid port too high", 65536, true},
		{"invalid port 80", 80, true},
		{"invalid port 443", 443, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lm.validatePort(tt.port)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "port must be between 1024 and 65535")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLifecycleManager_processInfoToInstance(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := DefaultInstanceConfig()
	configWriter, _ := NewConfigWriter(config)
	lm := NewLifecycleManager(nil, configWriter, logger)

	t.Run("converts running process info", func(t *testing.T) {
		info := &supervisord.ProcessInfo{
			Name:          "gowa_3001",
			StateName:     "RUNNING",
			Pid:           12345,
			Start:         int(time.Now().Add(-1 * time.Hour).Unix()),
			StdoutLogfile: "/var/log/gowa_3001.log",
			StderrLogfile: "/var/log/gowa_3001.err",
		}

		instance := lm.processInfoToInstance(info)
		assert.NotNil(t, instance)
		assert.Equal(t, 3001, instance.Port)
		assert.Equal(t, StateRunning, instance.State)
		assert.Equal(t, 12345, instance.PID)
		assert.True(t, instance.Uptime > 0)
		assert.Equal(t, "/var/log/gowa_3001.log", instance.LogFiles.Stdout)
		assert.Equal(t, "/var/log/gowa_3001.err", instance.LogFiles.Stderr)
	})

	t.Run("converts stopped process info", func(t *testing.T) {
		info := &supervisord.ProcessInfo{
			Name:      "gowa_3002",
			StateName: "STOPPED",
			Pid:       0,
		}

		instance := lm.processInfoToInstance(info)
		assert.NotNil(t, instance)
		assert.Equal(t, 3002, instance.Port)
		assert.Equal(t, StateStopped, instance.State)
		assert.Equal(t, 0, instance.PID)
	})

	t.Run("converts starting process info", func(t *testing.T) {
		info := &supervisord.ProcessInfo{
			Name:      "gowa_3003",
			StateName: "STARTING",
		}

		instance := lm.processInfoToInstance(info)
		assert.NotNil(t, instance)
		assert.Equal(t, StateStarting, instance.State)
	})

	t.Run("converts fatal process info", func(t *testing.T) {
		info := &supervisord.ProcessInfo{
			Name:      "gowa_3004",
			StateName: "FATAL",
		}

		instance := lm.processInfoToInstance(info)
		assert.NotNil(t, instance)
		assert.Equal(t, StateFatal, instance.State)
	})

	t.Run("converts unknown state process info", func(t *testing.T) {
		info := &supervisord.ProcessInfo{
			Name:      "gowa_3005",
			StateName: "BACKOFF",
		}

		instance := lm.processInfoToInstance(info)
		assert.NotNil(t, instance)
		assert.Equal(t, StateUnknown, instance.State)
	})

	t.Run("returns nil for invalid program name", func(t *testing.T) {
		info := &supervisord.ProcessInfo{
			Name:      "gowa_invalid",
			StateName: "RUNNING",
		}

		instance := lm.processInfoToInstance(info)
		assert.Nil(t, instance)
	})

	t.Run("handles zero start time", func(t *testing.T) {
		info := &supervisord.ProcessInfo{
			Name:      "gowa_3006",
			StateName: "STOPPED",
			Start:     0,
		}

		instance := lm.processInfoToInstance(info)
		assert.NotNil(t, instance)
		assert.Equal(t, time.Duration(0), instance.Uptime)
	})
}

func TestLifecycleManager_isPortAvailable(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := DefaultInstanceConfig()
	configWriter, _ := NewConfigWriter(config)
	lm := NewLifecycleManager(nil, configWriter, logger)

	t.Run("returns true for available port", func(t *testing.T) {
		// Port 59999 is unlikely to be in use
		available := lm.isPortAvailable(59999)
		assert.True(t, available)
	})

	// Note: Testing unavailable port would require binding a port first
	// which can be flaky in CI environments
}

func TestInstanceState(t *testing.T) {
	t.Run("state constants have correct values", func(t *testing.T) {
		assert.Equal(t, InstanceState("RUNNING"), StateRunning)
		assert.Equal(t, InstanceState("STOPPED"), StateStopped)
		assert.Equal(t, InstanceState("STARTING"), StateStarting)
		assert.Equal(t, InstanceState("FATAL"), StateFatal)
		assert.Equal(t, InstanceState("UNKNOWN"), StateUnknown)
	})
}

func TestInstance(t *testing.T) {
	t.Run("instance struct fields", func(t *testing.T) {
		instance := &Instance{
			Port:   3001,
			State:  StateRunning,
			PID:    12345,
			Uptime: 1 * time.Hour,
			LogFiles: LogFiles{
				Stdout: "/var/log/stdout.log",
				Stderr: "/var/log/stderr.log",
			},
		}

		assert.Equal(t, 3001, instance.Port)
		assert.Equal(t, StateRunning, instance.State)
		assert.Equal(t, 12345, instance.PID)
		assert.Equal(t, 1*time.Hour, instance.Uptime)
		assert.Equal(t, "/var/log/stdout.log", instance.LogFiles.Stdout)
		assert.Equal(t, "/var/log/stderr.log", instance.LogFiles.Stderr)
	})
}
