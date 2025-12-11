package admin

import (
	"time"

	"github.com/sirupsen/logrus"
)

// AuditLog represents an audit log entry for instance operations
type AuditLog struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Port      int    `json:"port,omitempty"`
	RequestID string `json:"request_id"`
	Result    string `json:"result"`
	Error     string `json:"error,omitempty"`
	Duration  string `json:"duration,omitempty"`
}

// AuditLogger handles audit logging for admin operations
type AuditLogger struct {
	logger *logrus.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *logrus.Logger) *AuditLogger {
	return &AuditLogger{logger: logger}
}

// LogOperation logs an instance operation with structured audit data
func (a *AuditLogger) LogOperation(action string, port int, requestID string, result string, err error, duration time.Duration) {
	entry := AuditLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    action,
		Port:      port,
		RequestID: requestID,
		Result:    result,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if duration > 0 {
		entry.Duration = duration.String()
	}

	a.logger.WithFields(logrus.Fields{
		"audit":      true,
		"timestamp":  entry.Timestamp,
		"action":     entry.Action,
		"port":       entry.Port,
		"request_id": entry.RequestID,
		"result":     entry.Result,
		"error":      entry.Error,
		"duration":   entry.Duration,
	}).Info("Audit event")
}

// LogCreate logs an instance creation audit event
func (a *AuditLogger) LogCreate(port int, requestID string, result string, err error, duration time.Duration) {
	a.LogOperation("create", port, requestID, result, err, duration)
}

// LogDelete logs an instance deletion audit event
func (a *AuditLogger) LogDelete(port int, requestID string, result string, err error, duration time.Duration) {
	a.LogOperation("delete", port, requestID, result, err, duration)
}

// LogUpdate logs an instance update audit event
func (a *AuditLogger) LogUpdate(port int, requestID string, result string, err error, duration time.Duration) {
	a.LogOperation("update", port, requestID, result, err, duration)
}
