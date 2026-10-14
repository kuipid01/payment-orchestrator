package webhooks

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// newTestLogger returns a debug-level logger writing into buf.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// The inbound pipeline trace is gated behind LOG_WEBHOOK_VERBOSE: off, the
// service emits nothing; on, it emits the step and its attributes.
func TestServiceTraceRespectsVerboseFlag(t *testing.T) {
	var quiet bytes.Buffer
	(&Service{Logger: newTestLogger(&quiet), Verbose: false}).
		tracef("webhook inbound received", "provider", "paystack", "connection_id", "conn_1")
	if quiet.Len() != 0 {
		t.Fatalf("Verbose=false emitted %q, want nothing", quiet.String())
	}

	var loud bytes.Buffer
	(&Service{Logger: newTestLogger(&loud), Verbose: true}).
		tracef("webhook inbound received", "provider", "paystack", "connection_id", "conn_1")
	out := loud.String()
	if !strings.Contains(out, "webhook inbound received") ||
		!strings.Contains(out, "paystack") || !strings.Contains(out, "conn_1") {
		t.Fatalf("Verbose=true emitted %q, want the step and its attributes", out)
	}
}

// A rejected signature is the one thing an operator must always see, so the
// verbose flag must not be able to hide it.
func TestServiceWarnIsNeverGated(t *testing.T) {
	var buf bytes.Buffer
	(&Service{Logger: newTestLogger(&buf), Verbose: false}).
		warn("webhook rejected: signature verification failed", "provider", "paystack")
	if !strings.Contains(buf.String(), "signature verification failed") {
		t.Fatalf("warn was suppressed by the verbose flag: %q", buf.String())
	}
}

// Neither the trace nor the warning path may panic when no logger is wired.
func TestServiceLoggingWithoutLogger(t *testing.T) {
	service := &Service{Verbose: true}
	service.tracef("webhook inbound received")
	service.warn("webhook rejected")
}
