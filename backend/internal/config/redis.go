package config

import "time"

// RedisConfig describes the optional Redis connection used for rate limiting,
// idempotency locks and caching. When URL is empty the process uses the
// in-process store (single-instance deployments only).
type RedisConfig struct {
	URL      string
	Prefix   string
	PoolSize int
	Timeout  time.Duration
}

// Enabled reports whether a shared Redis instance is configured.
func (r RedisConfig) Enabled() bool { return r.URL != "" }
