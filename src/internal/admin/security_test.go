package admin

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestValidateAndWarn(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("returns warning for default basic auth", func(t *testing.T) {
		config := &SecurityConfig{
			BasicAuth:     "admin:admin",
			WebhookSecret: "a-very-long-secret-for-testing-purposes",
			Logger:        logger,
		}

		warnings := ValidateAndWarn(config)
		assert.Len(t, warnings, 1)
		assert.Equal(t, "WEAK_BASIC_AUTH", warnings[0].Code)
		assert.Equal(t, "CRITICAL", warnings[0].Level)
	})

	t.Run("returns warning for weak webhook secret", func(t *testing.T) {
		config := &SecurityConfig{
			BasicAuth:     "admin:StrongPass123",
			WebhookSecret: "secret",
			Logger:        logger,
		}

		warnings := ValidateAndWarn(config)
		assert.Len(t, warnings, 1)
		assert.Equal(t, "WEAK_WEBHOOK_SECRET", warnings[0].Code)
	})

	t.Run("returns warning for short webhook secret", func(t *testing.T) {
		config := &SecurityConfig{
			BasicAuth:     "admin:StrongPass123",
			WebhookSecret: "short",
			Logger:        logger,
		}

		warnings := ValidateAndWarn(config)
		assert.Len(t, warnings, 1)
		assert.Equal(t, "WEAK_WEBHOOK_SECRET", warnings[0].Code)
	})

	t.Run("returns no warnings for strong credentials", func(t *testing.T) {
		config := &SecurityConfig{
			BasicAuth:     "admin:StrongPass123",
			WebhookSecret: "a-very-long-and-secure-webhook-secret-123",
			Logger:        logger,
		}

		warnings := ValidateAndWarn(config)
		assert.Empty(t, warnings)
	})
}

func TestIsWeakCredential(t *testing.T) {
	tests := []struct {
		name     string
		auth     string
		expected bool
	}{
		{"default admin:admin", "admin:admin", true},
		{"default admin:password", "admin:password", true},
		{"root:root", "root:root", true},
		{"test:test", "test:test", true},
		{"short password", "admin:abc", true},
		{"no complexity", "admin:abcdefgh", true},
		{"strong password", "admin:StrongPass123", false},
		{"complex password", "admin:MyP@ssw0rd", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWeakCredential(tt.auth)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasRequiredComplexity(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"all lowercase", "abcdefgh", false},
		{"all uppercase", "ABCDEFGH", false},
		{"all digits", "12345678", false},
		{"upper and lower", "ABCDabcd", true},
		{"lower and digit", "abcd1234", true},
		{"upper and digit", "ABCD1234", true},
		{"all three", "Abcd1234", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasRequiredComplexity(tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsProductionMode(t *testing.T) {
	t.Run("returns true for GO_ENV=production", func(t *testing.T) {
		os.Setenv("GO_ENV", "production")
		defer os.Unsetenv("GO_ENV")

		result := isProductionMode()
		assert.True(t, result)
	})

	t.Run("returns true for GO_ENV=prod", func(t *testing.T) {
		os.Setenv("GO_ENV", "prod")
		defer os.Unsetenv("GO_ENV")

		result := isProductionMode()
		assert.True(t, result)
	})

	t.Run("returns true for APP_ENV=production", func(t *testing.T) {
		os.Setenv("APP_ENV", "production")
		defer os.Unsetenv("APP_ENV")

		result := isProductionMode()
		assert.True(t, result)
	})

	t.Run("returns false when debug is enabled", func(t *testing.T) {
		os.Setenv("GOWA_DEBUG", "true")
		os.Setenv("ADMIN_DEBUG", "true")
		defer func() {
			os.Unsetenv("GOWA_DEBUG")
			os.Unsetenv("ADMIN_DEBUG")
		}()

		result := isProductionMode()
		assert.False(t, result)
	})
}

func TestLogDefaultCredentialWarnings(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("logs warning when GOWA_BASIC_AUTH is not set", func(t *testing.T) {
		os.Unsetenv("GOWA_BASIC_AUTH")
		os.Unsetenv("GOWA_WEBHOOK_SECRET")

		// Should not panic
		LogDefaultCredentialWarnings(logger)
	})
}

func TestValidateAndWarn_ProductionMode(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Save original environment
	originalGoEnv := os.Getenv("GO_ENV")
	originalAppEnv := os.Getenv("APP_ENV")
	originalDebug := os.Getenv("GOWA_DEBUG")
	originalAdminDebug := os.Getenv("ADMIN_DEBUG")

	defer func() {
		if originalGoEnv != "" {
			os.Setenv("GO_ENV", originalGoEnv)
		} else {
			os.Unsetenv("GO_ENV")
		}
		if originalAppEnv != "" {
			os.Setenv("APP_ENV", originalAppEnv)
		} else {
			os.Unsetenv("APP_ENV")
		}
		if originalDebug != "" {
			os.Setenv("GOWA_DEBUG", originalDebug)
		} else {
			os.Unsetenv("GOWA_DEBUG")
		}
		if originalAdminDebug != "" {
			os.Setenv("ADMIN_DEBUG", originalAdminDebug)
		} else {
			os.Unsetenv("ADMIN_DEBUG")
		}
	}()

	t.Run("logs error in production mode with warnings", func(t *testing.T) {
		os.Setenv("GO_ENV", "production")
		os.Unsetenv("GOWA_DEBUG")
		os.Unsetenv("ADMIN_DEBUG")

		config := &SecurityConfig{
			BasicAuth:     "admin:admin", // Weak credential to trigger warning
			WebhookSecret: "a-very-long-and-secure-webhook-secret-123",
			Logger:        logger,
		}

		warnings := ValidateAndWarn(config)
		assert.NotEmpty(t, warnings)
	})
}

func TestIsProductionMode_DebugVariants(t *testing.T) {
	// Save original environment
	originalGoEnv := os.Getenv("GO_ENV")
	originalAppEnv := os.Getenv("APP_ENV")
	originalDebug := os.Getenv("GOWA_DEBUG")
	originalAdminDebug := os.Getenv("ADMIN_DEBUG")

	defer func() {
		if originalGoEnv != "" {
			os.Setenv("GO_ENV", originalGoEnv)
		} else {
			os.Unsetenv("GO_ENV")
		}
		if originalAppEnv != "" {
			os.Setenv("APP_ENV", originalAppEnv)
		} else {
			os.Unsetenv("APP_ENV")
		}
		if originalDebug != "" {
			os.Setenv("GOWA_DEBUG", originalDebug)
		} else {
			os.Unsetenv("GOWA_DEBUG")
		}
		if originalAdminDebug != "" {
			os.Setenv("ADMIN_DEBUG", originalAdminDebug)
		} else {
			os.Unsetenv("ADMIN_DEBUG")
		}
	}()

	t.Run("returns true when debug is empty and admin debug is not true", func(t *testing.T) {
		os.Unsetenv("GO_ENV")
		os.Unsetenv("APP_ENV")
		os.Unsetenv("GOWA_DEBUG")
		os.Unsetenv("ADMIN_DEBUG")

		result := isProductionMode()
		assert.True(t, result)
	})

	t.Run("returns true when debug is false", func(t *testing.T) {
		os.Unsetenv("GO_ENV")
		os.Unsetenv("APP_ENV")
		os.Setenv("GOWA_DEBUG", "false")
		os.Unsetenv("ADMIN_DEBUG")

		result := isProductionMode()
		assert.True(t, result)
	})

	t.Run("returns false when debug is true", func(t *testing.T) {
		os.Unsetenv("GO_ENV")
		os.Unsetenv("APP_ENV")
		os.Setenv("GOWA_DEBUG", "true")
		os.Unsetenv("ADMIN_DEBUG")

		result := isProductionMode()
		assert.False(t, result)
	})
}
