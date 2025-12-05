package admin

import (
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewAuditLogger(t *testing.T) {
	logger := logrus.New()
	auditLogger := NewAuditLogger(logger)
	assert.NotNil(t, auditLogger)
}

func TestAuditLogger_LogOperation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	auditLogger := NewAuditLogger(logger)

	t.Run("logs successful operation", func(t *testing.T) {
		auditLogger.LogOperation("create", 3001, "req-123", "success", nil, 100*time.Millisecond)
	})

	t.Run("logs failed operation with error", func(t *testing.T) {
		auditLogger.LogOperation("create", 3001, "req-456", "failed", errors.New("test error"), 50*time.Millisecond)
	})

	t.Run("logs operation without duration", func(t *testing.T) {
		auditLogger.LogOperation("delete", 3002, "req-789", "success", nil, 0)
	})
}

func TestAuditLogger_LogCreate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	auditLogger := NewAuditLogger(logger)

	t.Run("logs create operation", func(t *testing.T) {
		auditLogger.LogCreate(3001, "req-123", "success", nil, 100*time.Millisecond)
	})

	t.Run("logs failed create operation", func(t *testing.T) {
		auditLogger.LogCreate(3001, "req-456", "failed", errors.New("port in use"), 50*time.Millisecond)
	})
}

func TestAuditLogger_LogDelete(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	auditLogger := NewAuditLogger(logger)

	t.Run("logs delete operation", func(t *testing.T) {
		auditLogger.LogDelete(3001, "req-123", "success", nil, 100*time.Millisecond)
	})

	t.Run("logs failed delete operation", func(t *testing.T) {
		auditLogger.LogDelete(3001, "req-456", "failed", errors.New("not found"), 50*time.Millisecond)
	})
}

func TestAuditLogger_LogUpdate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	auditLogger := NewAuditLogger(logger)

	t.Run("logs update operation", func(t *testing.T) {
		auditLogger.LogUpdate(3001, "req-123", "success", nil, 100*time.Millisecond)
	})

	t.Run("logs failed update operation", func(t *testing.T) {
		auditLogger.LogUpdate(3001, "req-456", "failed", errors.New("not found"), 50*time.Millisecond)
	})
}
