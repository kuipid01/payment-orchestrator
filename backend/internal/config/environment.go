// Package config loads and validates every environment-driven setting exactly
// once at startup. Nothing else in the application reads os.Getenv.
package config

import (
	"github.com/kuipid01/payconnect/internal/constants"
)

// IsProduction reports whether the process runs in production.
func (c Config) IsProduction() bool {
	return c.App.Environment == constants.EnvProduction
}

// IsDevelopment reports whether the process runs in development.
func (c Config) IsDevelopment() bool {
	return c.App.Environment == constants.EnvDevelopment
}

// BaseURL returns the public origin this API is reachable at.
func (c Config) BaseURL() string { return c.App.BaseURL }
