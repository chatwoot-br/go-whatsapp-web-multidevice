package admin

import (
	"os"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
)

// SecurityConfig holds security-related configuration for validation
type SecurityConfig struct {
	BasicAuth     string
	WebhookSecret string
	Logger        *logrus.Logger
}

// SecurityWarning represents a security warning
type SecurityWarning struct {
	Level   string // "CRITICAL", "HIGH", "MEDIUM"
	Code    string
	Message string
}

// ValidateAndWarn checks for weak credentials and logs warnings
// Returns a list of security warnings found
func ValidateAndWarn(config *SecurityConfig) []SecurityWarning {
	var warnings []SecurityWarning

	// Check BasicAuth
	if config.BasicAuth != "" {
		if isWeakCredential(config.BasicAuth) {
			w := SecurityWarning{
				Level:   "CRITICAL",
				Code:    "WEAK_BASIC_AUTH",
				Message: "GOWA_BASIC_AUTH uses weak default credentials 'admin:admin'. Set a strong password for production use.",
			}
			warnings = append(warnings, w)
			if config.Logger != nil {
				config.Logger.Warn("[SECURITY] " + w.Message)
			}
		}
	}

	// Check WebhookSecret
	if config.WebhookSecret == "secret" || len(config.WebhookSecret) < 16 {
		w := SecurityWarning{
			Level:   "HIGH",
			Code:    "WEAK_WEBHOOK_SECRET",
			Message: "GOWA_WEBHOOK_SECRET is weak or uses default value. Set a random secret (32+ characters) for production.",
		}
		warnings = append(warnings, w)
		if config.Logger != nil {
			config.Logger.Warn("[SECURITY] " + w.Message)
		}
	}

	// Check if running in production mode
	if isProductionMode() && len(warnings) > 0 {
		if config.Logger != nil {
			config.Logger.Error("[SECURITY] Production mode detected with security warnings. Review configuration immediately!")
		}
	}

	return warnings
}

// LogDefaultCredentialWarnings logs warnings if default credentials are being used
func LogDefaultCredentialWarnings(logger *logrus.Logger) {
	// Check if GOWA_BASIC_AUTH is using default
	if os.Getenv("GOWA_BASIC_AUTH") == "" {
		logger.Warn("[CONFIG] GOWA_BASIC_AUTH not set, using default 'admin:admin'. Set a strong password for production.")
	}

	// Check if GOWA_WEBHOOK_SECRET is using default
	if os.Getenv("GOWA_WEBHOOK_SECRET") == "" {
		logger.Warn("[CONFIG] GOWA_WEBHOOK_SECRET not set, using default 'secret'. Set a strong secret for production.")
	}
}

// isWeakCredential checks if the credential is a known weak default
func isWeakCredential(auth string) bool {
	weakDefaults := []string{
		"admin:admin",
		"admin:password",
		"admin:123456",
		"root:root",
		"user:user",
		"test:test",
	}

	lower := strings.ToLower(auth)
	for _, weak := range weakDefaults {
		if lower == weak {
			return true
		}
	}

	// Check password strength (after colon)
	parts := strings.SplitN(auth, ":", 2)
	if len(parts) == 2 {
		password := parts[1]
		if len(password) < 8 {
			return true
		}
		if !hasRequiredComplexity(password) {
			return true
		}
	}

	return false
}

// hasRequiredComplexity checks for minimum password complexity
func hasRequiredComplexity(password string) bool {
	var hasUpper, hasLower, hasDigit bool

	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	// Require at least 2 of 3 character types for minimal complexity
	complexity := 0
	if hasUpper {
		complexity++
	}
	if hasLower {
		complexity++
	}
	if hasDigit {
		complexity++
	}

	return complexity >= 2
}

// isProductionMode detects if running in production
func isProductionMode() bool {
	// Check common production indicators
	env := strings.ToLower(os.Getenv("GO_ENV"))
	if env == "production" || env == "prod" {
		return true
	}

	env = strings.ToLower(os.Getenv("APP_ENV"))
	if env == "production" || env == "prod" {
		return true
	}

	// Check if debug is disabled (production indicator)
	debug := os.Getenv("GOWA_DEBUG")
	if debug == "" || debug == "false" {
		// Could be production
		return os.Getenv("ADMIN_DEBUG") != "true"
	}

	return false
}
