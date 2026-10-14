package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

// emit runs one query trace through the production SQL logger and returns
// everything it wrote.
func emit(err error, elapsed time.Duration) string {
	var buf bytes.Buffer
	newSQLLogger(&buf).Trace(context.Background(), time.Now().Add(-elapsed),
		func() (string, int64) {
			return "SELECT * FROM \"provider_connections\" WHERE id = 'conn_x' LIMIT 1", 0
		}, err)
	return buf.String()
}

// A lookup that finds nothing is a normal outcome, not an incident: the caller
// already logs it with context. Emitting GORM's own error line for it is what
// made the logs unusable, so it must stay silent — including when wrapped.
func TestSQLLoggerSilencesRecordNotFound(t *testing.T) {
	if out := emit(gorm.ErrRecordNotFound, 3*time.Millisecond); out != "" {
		t.Fatalf("record-not-found logged %q, want nothing", out)
	}
	wrapped := fmt.Errorf("providers: lookup: %w", gorm.ErrRecordNotFound)
	if out := emit(wrapped, 3*time.Millisecond); out != "" {
		t.Fatalf("wrapped record-not-found logged %q, want nothing", out)
	}
}

// A fast, successful query is routine and must not appear at the warn level.
func TestSQLLoggerIgnoresRoutineQueries(t *testing.T) {
	if out := emit(nil, 3*time.Millisecond); out != "" {
		t.Fatalf("routine query logged %q, want nothing", out)
	}
}

// Silence must be specific to "not found": a genuine failure still surfaces.
func TestSQLLoggerKeepsRealErrors(t *testing.T) {
	out := emit(errors.New("connection reset by peer"), 3*time.Millisecond)
	if !strings.Contains(out, "connection reset by peer") {
		t.Fatalf("real error was suppressed: %q", out)
	}
}

// A slow query is the signal the default logger exists for; it must survive.
func TestSQLLoggerKeepsSlowQueries(t *testing.T) {
	out := emit(nil, 2*time.Second)
	if !strings.Contains(out, "SLOW") {
		t.Fatalf("slow query was not reported: %q", out)
	}
}
