package config

import (
	"fmt"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
)

// Validate returns every configuration problem found, so a misconfigured
// deployment fails loudly at startup instead of at the first request.
func (c Config) Validate() error {
	var problems []string

	if !constants.IsValidAppEnvironment(c.App.Environment) {
		problems = append(problems, fmt.Sprintf("APP_ENV %q must be one of development|test|production", c.App.Environment))
	}
	if strings.TrimSpace(c.Server.Port) == "" {
		problems = append(problems, "PORT must not be empty")
	}
	if strings.TrimSpace(c.Database.DSN()) == "" {
		problems = append(problems, "DATABASE_URL or DATABASE_HOST/DATABASE_NAME must be set")
	}
	if strings.TrimSpace(c.Security.EncryptionKey) == "" {
		problems = append(problems, "ENCRYPTION_KEY must be set (32 bytes)")
	}
	if strings.TrimSpace(c.Security.APIKeyPrefix) == "" {
		problems = append(problems, "API_KEY_PREFIX must not be empty")
	}
	if c.IsProduction() {
		if len(c.Security.JWTSecret) < 32 {
			problems = append(problems, "JWT_SECRET must be at least 32 characters in production")
		}
		if len(c.Security.EncryptionKey) < 16 {
			problems = append(problems, "ENCRYPTION_KEY must be at least 16 bytes in production")
		}
		if c.Security.EncryptionKey == "payconnect-dev-encryption-key-32b!" {
			problems = append(problems, "ENCRYPTION_KEY must not be the development default in production")
		}
		for _, origin := range c.Security.AllowedOrigins {
			if strings.Contains(origin, "localhost") {
				problems = append(problems, "ALLOWED_ORIGINS must not contain localhost in production")
			}
		}
	}
	if !isValidLogLevel(c.Log.Level) {
		problems = append(problems, fmt.Sprintf("LOG_LEVEL %q must be one of debug|info|warn|error", c.Log.Level))
	}
	if c.Providers.MaxRetries < 0 {
		problems = append(problems, "PROVIDER_MAX_RETRIES must not be negative")
	}
	if c.Limits.AuthMaxFailures < 0 {
		problems = append(problems, "AUTH_MAX_FAILURES must not be negative")
	}
	if c.Limits.AuthLockoutDuration < 0 {
		problems = append(problems, "AUTH_LOCKOUT_DURATION must not be negative")
	}
	if c.Limits.AuthMaxFailures > 0 && c.Limits.AuthLockoutDuration == 0 {
		problems = append(problems, "AUTH_LOCKOUT_DURATION must be greater than zero when AUTH_MAX_FAILURES is set")
	}
	if c.IsProduction() && c.Limits.AuthMaxFailures == 0 {
		problems = append(problems, "AUTH_MAX_FAILURES must be greater than zero in production")
	}

	// A gate name that is not recognised must not degrade to log-only in
	// silence: whoever wrote BILLING_ENFORCED would believe the limit is
	// enforced when it is not.
	for _, gate := range c.Billing.EnforcedGates {
		if !constants.IsEnforceableGate(gate) {
			problems = append(problems, fmt.Sprintf(
				"BILLING_ENFORCED names unknown gate %q (known: %s)",
				gate, strings.Join(constants.EnforceableGates(), ", ")))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// isValidLogLevel reports whether level names a level the logger understands.
func isValidLogLevel(level string) bool {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "info", "warn", "warning", "error":
		return true
	}
	return false
}
