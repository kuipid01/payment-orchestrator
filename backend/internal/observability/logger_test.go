package observability

import (
	"log/slog"
	"testing"
)

// ParseLevel drives LOG_LEVEL. An unknown or empty value must fall back to the
// environment default rather than silencing every log line.
func TestParseLevel(t *testing.T) {
	cases := []struct {
		level       string
		environment string
		want        slog.Level
	}{
		{"debug", "production", slog.LevelDebug},
		{"info", "development", slog.LevelInfo},
		{"warn", "development", slog.LevelWarn},
		{"WARNING", "development", slog.LevelWarn},
		{"error", "development", slog.LevelError},
		{" debug ", "development", slog.LevelDebug},
		{"", "production", slog.LevelInfo},
		{"", "development", slog.LevelDebug},
		{"nonsense", "production", slog.LevelInfo},
		{"nonsense", "development", slog.LevelDebug},
	}
	for _, c := range cases {
		if got := ParseLevel(c.level, c.environment); got != c.want {
			t.Errorf("ParseLevel(%q, %q) = %v, want %v", c.level, c.environment, got, c.want)
		}
	}
}
