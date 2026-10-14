package observability

import (
	"log/slog"
	"time"
)

// Span is a placeholder tracing primitive. It records start/end timestamps and
// emits a structured log line, which is enough to correlate a request across
// handler → service → provider before a real tracer is wired in.
type Span struct {
	name  string
	start time.Time
	log   *slog.Logger
}

// StartSpan opens a span for the given operation.
func StartSpan(logger *slog.Logger, name string) *Span {
	return &Span{name: name, start: time.Now(), log: logger}
}

// End closes the span and logs its duration.
func (s *Span) End(err error) {
	if s == nil {
		return
	}
	attrs := []any{"span", s.name, "duration_ms", time.Since(s.start).Milliseconds()}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	if s.log != nil {
		s.log.Info("span finished", attrs...)
	}
}
