// Package observability owns logging, metrics and tracing setup.
package observability

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

// NewLogger builds the application logger with the environment's default level:
// JSON in production, human-readable text elsewhere. Every log line carries the
// service name.
func NewLogger(environment, service string) *slog.Logger {
	return NewLoggerWithLevel(environment, service, "")
}

// NewLoggerWithLevel builds the logger with an explicit minimum level. An empty
// or unrecognised level falls back to the environment default.
func NewLoggerWithLevel(environment, service, level string) *slog.Logger {
	minLevel := ParseLevel(level, environment)
	opts := &slog.HandlerOptions{Level: minLevel}

	var handler slog.Handler
	if environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		// Use tint for a gorgeous, colorized, readable local terminal output
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      minLevel,
			TimeFormat: time.Kitchen, // Keeps timestamps short (e.g., 12:11PM)
		})
	}
	return slog.New(handler).With("service", service, "env", environment)
}

// ParseLevel maps a level name onto an slog level, falling back to the
// environment default so a typo can never silence every log line.
func ParseLevel(level, environment string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}
	if environment == "production" {
		return slog.LevelInfo
	}
	return slog.LevelDebug
}

// SetDefault installs the logger process-wide.
func SetDefault(logger *slog.Logger) { slog.SetDefault(logger) }

// Redact masks anything that looks like a credential before it reaches a log
// line. Call this on any value that originates from a client header or body.
func Redact(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:8] + "…(" + itoa(len(value)) + ")"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
