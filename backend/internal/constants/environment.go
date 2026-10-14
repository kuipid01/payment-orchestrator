// Package constants holds domain-level values shared across the application.
// No business logic, no configuration — just the vocabulary of the domain.
package constants

// Application environment: how the process is running. Distinct from the
// payment environment, which describes how money moves.
const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"
)

// Payment environments. A merchant may hold credentials for both.
const (
	EnvironmentTest = "test"
	EnvironmentLive = "live"
)

// IsValidAppEnvironment reports whether v is a known application environment.
func IsValidAppEnvironment(v string) bool {
	switch v {
	case EnvDevelopment, EnvTest, EnvProduction:
		return true
	}
	return false
}

// IsValidPaymentEnvironment reports whether v is a valid payment environment.
func IsValidPaymentEnvironment(v string) bool {
	return v == EnvironmentTest || v == EnvironmentLive
}
