package webhooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/events"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Backoff must grow monotonically and stay bounded, otherwise a broken merchant
// endpoint either gets hammered or is retried forever.
func TestBackoffGrowsAndIsCapped(t *testing.T) {
	previous := time.Duration(0)
	for attempt := 1; attempt <= 6; attempt++ {
		delay := backoff(attempt)
		if delay < previous {
			t.Fatalf("backoff(%d) = %s is shorter than the previous delay %s", attempt, delay, previous)
		}
		previous = delay
	}
	if backoff(1) != 30*time.Second {
		t.Errorf("first retry delay = %s, want 30s", backoff(1))
	}
	if backoff(50) > time.Hour {
		t.Errorf("backoff must be capped at 1h, got %s", backoff(50))
	}
	// A zero/negative attempt count must still produce a sane delay.
	if backoff(0) <= 0 {
		t.Errorf("backoff(0) = %s, want a positive delay", backoff(0))
	}
}

func TestEnvelopeCarriesEventData(t *testing.T) {
	event := events.New(constants.EventPaymentSucceeded, "mch_1", constants.EnvironmentTest)
	event.PaymentID = "pay_1"
	event.Reference = "ORDER-1"
	event.Data = json.RawMessage(`{"id":"pay_1","status":"successful"}`)

	body := envelope(event, "whd_1")
	if body["id"] != event.ID || body["type"] != constants.EventPaymentSucceeded {
		t.Errorf("envelope identity is wrong: %+v", body)
	}
	data, ok := body["data"].(map[string]any)
	if !ok || data["status"] != "successful" {
		t.Errorf("envelope data = %+v, want the payment payload", body["data"])
	}
	if _, err := time.Parse(time.RFC3339, body["created_at"].(string)); err != nil {
		t.Errorf("created_at is not RFC3339: %v", body["created_at"])
	}
	meta, ok := body["meta"].(map[string]any)
	if !ok {
		t.Fatalf("meta = %T, want a map so consumers have a stable place for delivery facts", body["meta"])
	}
	if meta["webhook_version"] != constants.WebhookPayloadVersion {
		t.Errorf("meta.webhook_version = %v, want %q", meta["webhook_version"], constants.WebhookPayloadVersion)
	}
	if meta["environment"] != constants.EnvironmentTest {
		t.Errorf("meta.environment = %v, want %q", meta["environment"], constants.EnvironmentTest)
	}
	if meta["delivery_id"] != "whd_1" {
		t.Errorf("meta.delivery_id = %v, want the delivery row id", meta["delivery_id"])
	}
}

// The contract is one shape for every provider and every event: exactly these
// top-level keys, always. A refund and a payment differ inside `data` only.
func TestEnvelopeShapeIsCanonical(t *testing.T) {
	for _, eventType := range []string{constants.EventPaymentSucceeded, constants.EventRefundSucceeded} {
		event := events.New(eventType, "mch_1", constants.EnvironmentLive)
		body := envelope(event, "whd_1")
		want := []string{"id", "type", "created_at", "meta", "data"}
		if len(body) != len(want) {
			t.Errorf("%s: envelope has %d keys %v, want %d: %v", eventType, len(body), body, len(want), want)
		}
		for _, key := range want {
			if _, ok := body[key]; !ok {
				t.Errorf("%s: envelope is missing %q", eventType, key)
			}
		}
	}
}

// An event without a payload still yields a well-formed body rather than null.
func TestEnvelopeWithoutData(t *testing.T) {
	event := events.New(constants.EventPaymentFailed, "mch_1", constants.EnvironmentTest)
	event.PaymentID = "pay_2"
	event.Reference = "ORDER-2"

	body := envelope(event, "whd_2")
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("data = %T, want a map", body["data"])
	}
	if data["payment_id"] != "pay_2" || data["reference"] != "ORDER-2" {
		t.Errorf("fallback data = %+v", data)
	}
}

// The webhook trace is gated behind LOG_WEBHOOK_VERBOSE: a quiet dispatcher emits
// nothing, and a verbose one emits the step and its attributes.
func TestDispatcherDebugRespectsVerboseFlag(t *testing.T) {
	var quiet bytes.Buffer
	quietLogger := slog.New(slog.NewTextHandler(&quiet, &slog.HandlerOptions{Level: slog.LevelDebug}))
	(&Dispatcher{Log: quietLogger, Verbose: false}).debug("webhook delivered", "delivery_id", "whd_1")
	if quiet.Len() != 0 {
		t.Fatalf("Verbose=false emitted %q, want nothing", quiet.String())
	}

	var loud bytes.Buffer
	loudLogger := slog.New(slog.NewTextHandler(&loud, &slog.HandlerOptions{Level: slog.LevelDebug}))
	(&Dispatcher{Log: loudLogger, Verbose: true}).debug("webhook delivered", "delivery_id", "whd_1")
	if !strings.Contains(loud.String(), "webhook delivered") || !strings.Contains(loud.String(), "whd_1") {
		t.Fatalf("Verbose=true emitted %q, want the step and its attributes", loud.String())
	}
}

// A verbose dispatcher with no logger must not panic.
func TestDispatcherDebugWithoutLogger(t *testing.T) {
	(&Dispatcher{Verbose: true}).debug("webhook delivered")
}

// Signature detection is used only for a log field, so it must never mistake an
// ordinary content-type for a signature.
func TestHasSignatureHeader(t *testing.T) {
	if hasSignatureHeader(map[string]string{"Content-Type": "application/json"}) {
		t.Error("content-type alone must not look like a signature header")
	}
	for _, header := range []string{"X-Paystack-Signature", "verif-hash", "x-moniepoint-signature", "x-signature"} {
		if !hasSignatureHeader(map[string]string{header: "abc"}) {
			t.Errorf("%s should be detected as a signature header", header)
		}
	}
}

func TestEndpointSubscribes(t *testing.T) {
	all := &Endpoint{}
	if !all.Subscribes(constants.EventPaymentSucceeded) {
		t.Error("an endpoint with no event filter should receive every event")
	}
	some := &Endpoint{Events: gormx.JSONB[[]string]{Data: []string{constants.EventPaymentSucceeded}}}
	if !some.Subscribes(constants.EventPaymentSucceeded) {
		t.Error("subscribed event was not delivered")
	}
	if some.Subscribes(constants.EventRefundSucceeded) {
		t.Error("unsubscribed event must not be delivered")
	}
}

// A short reply is stored verbatim: it is the merchant's only clue about why
// their endpoint rejected a delivery.
func TestResponseSnippetKeepsShortBody(t *testing.T) {
	const body = `{"error":"missing field amount"}`
	if got := responseSnippet(strings.NewReader(body)); got != body {
		t.Errorf("responseSnippet = %q, want the body unchanged", got)
	}
}

// A hostile or merely verbose endpoint cannot fill the table.
func TestResponseSnippetTruncatesLongBody(t *testing.T) {
	got := responseSnippet(strings.NewReader(strings.Repeat("a", ResponseBodyLimit*3)))
	if !strings.HasSuffix(got, "(truncated)") {
		t.Fatalf("long body was not marked truncated: %q", got)
	}
	if len(got) > ResponseBodyLimit+len("\n… (truncated)") {
		t.Errorf("stored %d bytes, want at most %d", len(got), ResponseBodyLimit+len("\n… (truncated)"))
	}
}

// Truncation lands wherever the limit falls, including mid-character; the value
// must stay valid UTF-8 or the delivery row cannot be serialized to a client.
func TestResponseSnippetStaysValidUTF8(t *testing.T) {
	body := strings.Repeat("a", ResponseBodyLimit-1) + "€"
	got := responseSnippet(strings.NewReader(body))
	if !utf8.ValidString(got) {
		t.Fatalf("truncated body is not valid UTF-8: %q", got)
	}
	if !strings.HasPrefix(got, "aaa") {
		t.Errorf("truncated body lost its head: %q", got)
	}
}

// Reading the body is best-effort: it must never be the reason an attempt fails.
func TestResponseSnippetIgnoresReadError(t *testing.T) {
	if got := responseSnippet(errorReader{}); got != "" {
		t.Errorf("responseSnippet = %q, want empty when the body cannot be read", got)
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }
