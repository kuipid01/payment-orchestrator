package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

func newTestAdapter(t *testing.T, handler http.HandlerFunc) *Adapter {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return New(Config{BaseURL: server.URL})
}

func testCreds() providers.Credentials {
	return providers.Credentials{
		SecretKey:   "sk_test_1234567890abcdef",
		Environment: constants.EnvironmentTest,
	}
}

func TestTestCredentialsUsesSecretKey(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/balance" {
			t.Errorf("path = %q, want /v1/balance", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testCreds().SecretKey {
			t.Errorf("Authorization = %q", got)
		}
		w.Write([]byte(`{"object":"balance","livemode":false}`))
	})

	if err := adapter.TestCredentials(context.Background(), testCreds()); err != nil {
		t.Fatalf("TestCredentials() = %v", err)
	}
}

func TestCreatePaymentCreatesHostedCheckoutSession(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions" || r.Method != http.MethodPost {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "checkout:pay-ref-1" {
			t.Errorf("Idempotency-Key = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.PostForm.Get("line_items[0][price_data][unit_amount]"); got != "2500" {
			t.Errorf("unit_amount = %q", got)
		}
		if got := r.PostForm.Get("line_items[0][price_data][currency]"); got != "usd" {
			t.Errorf("currency = %q", got)
		}
		if got := r.PostForm.Get("success_url"); got != "https://store.example/return?session_id={CHECKOUT_SESSION_ID}" {
			t.Errorf("success_url = %q", got)
		}
		if got := r.PostForm.Get("payment_intent_data[metadata][reference]"); got != "pay-ref-1" {
			t.Errorf("payment_intent metadata = %q", got)
		}
		w.Write([]byte(`{"id":"cs_test_123","object":"checkout.session","status":"open","payment_status":"unpaid","amount_total":2500,"currency":"usd","url":"https://checkout.stripe.com/c/pay/cs_test_123","client_reference_id":"pay-ref-1"}`))
	})

	result, err := adapter.CreatePayment(context.Background(), testCreds(), providers.PaymentRequest{
		Reference:   "pay-ref-1",
		AmountMinor: 2500,
		Currency:    "USD",
		Email:       "payer@example.com",
		Description: "Order 42",
		CallbackURL: "https://store.example/return",
		Metadata:    map[string]any{"order_id": "42"},
	})
	if err != nil {
		t.Fatalf("CreatePayment() = %v", err)
	}
	if result.ProviderReference != "cs_test_123" {
		t.Errorf("ProviderReference = %q", result.ProviderReference)
	}
	if result.AuthorizationURL != "https://checkout.stripe.com/c/pay/cs_test_123" {
		t.Errorf("AuthorizationURL = %q", result.AuthorizationURL)
	}
	if result.Status != string(constants.PaymentPending) {
		t.Errorf("Status = %q, want pending", result.Status)
	}
	if result.AmountMinor != 2500 || result.Currency != "USD" {
		t.Errorf("amount/currency = %d/%s", result.AmountMinor, result.Currency)
	}
}

func TestCreatePaymentRequiresCallbackURL(t *testing.T) {
	adapter := New(Config{})
	_, err := adapter.CreatePayment(context.Background(), testCreds(), providers.PaymentRequest{
		Reference:   "pay-ref-1",
		AmountMinor: 2500,
		Currency:    "USD",
	})
	if !apperrors.Is(err, "checkout_callback_required") {
		t.Fatalf("CreatePayment() error = %v, want callback_url validation", err)
	}
}

func TestVerifyPaymentMapsPaidSession(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions/cs_test_123" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`{"id":"cs_test_123","status":"complete","payment_status":"paid","amount_total":2500,"currency":"usd","payment_intent":"pi_123","client_reference_id":"pay-ref-1"}`))
	})

	result, err := adapter.VerifyPayment(context.Background(), testCreds(), "cs_test_123")
	if err != nil {
		t.Fatalf("VerifyPayment() = %v", err)
	}
	if result.Status != string(constants.PaymentSuccessful) {
		t.Errorf("Status = %q, want successful", result.Status)
	}
	if result.ProviderReference != "cs_test_123" {
		t.Errorf("ProviderReference = %q", result.ProviderReference)
	}
}

func TestRefundPaymentResolvesSessionPaymentIntent(t *testing.T) {
	step := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			if r.URL.Path != "/v1/checkout/sessions/cs_test_123" {
				t.Errorf("session path = %q", r.URL.Path)
			}
			w.Write([]byte(`{"id":"cs_test_123","payment_intent":"pi_123","status":"complete","payment_status":"paid","amount_total":2500,"currency":"usd"}`))
		case 2:
			if r.URL.Path != "/v1/refunds" {
				t.Errorf("refund path = %q", r.URL.Path)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if got := r.PostForm.Get("payment_intent"); got != "pi_123" {
				t.Errorf("payment_intent = %q", got)
			}
			if got := r.PostForm.Get("amount"); got != "1000" {
				t.Errorf("amount = %q", got)
			}
			w.Write([]byte(`{"id":"re_123","status":"succeeded","amount":1000,"currency":"usd"}`))
		}
	})

	result, err := adapter.RefundPayment(context.Background(), testCreds(), providers.RefundRequest{
		PaymentReference:  "pay-ref-1",
		ProviderReference: "cs_test_123",
		AmountMinor:       1000,
		Currency:          "USD",
	})
	if err != nil {
		t.Fatalf("RefundPayment() = %v", err)
	}
	if result.ProviderRefundID != "re_123" || result.Status != string(constants.RefundSuccessful) {
		t.Fatalf("RefundPayment() = %#v", result)
	}
	if result.AmountMinor != 1000 {
		t.Errorf("AmountMinor = %d", result.AmountMinor)
	}
}

func TestVerifyWebhookAcceptsSignedSessionEvent(t *testing.T) {
	body := []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"id":"cs_test_123","status":"complete","payment_status":"paid","amount_total":2500,"currency":"usd","client_reference_id":"pay-ref-1"}}}`)
	signature := sign("whsec_test", body, time.Now())

	result, err := New(Config{}).VerifyWebhook(providers.WebhookVerificationRequest{
		Headers:       map[string]string{"Stripe-Signature": signature},
		Body:          body,
		WebhookSecret: "whsec_test",
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() = %v", err)
	}
	if !result.SignatureValid || result.EventID != "stripe:evt_1" {
		t.Fatalf("VerifyWebhook() = %#v", result)
	}
	if result.Type != constants.EventPaymentSucceeded || result.Status != string(constants.PaymentSuccessful) {
		t.Errorf("type/status = %s/%s", result.Type, result.Status)
	}
	if result.Reference != "pay-ref-1" || result.ProviderReference != "cs_test_123" || result.AmountMinor != 2500 {
		t.Errorf("refs/amount = %q/%q/%d", result.Reference, result.ProviderReference, result.AmountMinor)
	}
}

func TestVerifyWebhookRejectsTamperedBody(t *testing.T) {
	body := []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"id":"cs_test_123","payment_status":"paid"}}}`)
	signature := sign("whsec_test", body, time.Now())

	_, err := New(Config{}).VerifyWebhook(providers.WebhookVerificationRequest{
		Headers:       map[string]string{"Stripe-Signature": signature},
		Body:          []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"id":"cs_other","payment_status":"paid"}}}`),
		WebhookSecret: "whsec_test",
	})
	if !apperrors.Is(err, apperrors.CodeWebhookSignatureInvalid) {
		t.Fatalf("VerifyWebhook() error = %v, want signature rejection", err)
	}
}

func TestVerifyWebhookRejectsStaleSignature(t *testing.T) {
	body := []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"id":"cs_test_123","payment_status":"paid"}}}`)
	signature := sign("whsec_test", body, time.Now().Add(-10*time.Minute))

	_, err := New(Config{}).VerifyWebhook(providers.WebhookVerificationRequest{
		Headers:       map[string]string{"Stripe-Signature": signature},
		Body:          body,
		WebhookSecret: "whsec_test",
	})
	if !apperrors.Is(err, apperrors.CodeWebhookSignatureInvalid) {
		t.Fatalf("VerifyWebhook() error = %v, want replay rejection", err)
	}
}

func TestVerifyWebhookMapsAsyncFailureToFailed(t *testing.T) {
	body := []byte(`{"id":"evt_3","type":"checkout.session.async_payment_failed","data":{"object":{"id":"cs_test_123","status":"complete","payment_status":"unpaid","amount_total":2500,"currency":"usd","client_reference_id":"pay-ref-1"}}}`)
	signature := sign("whsec_test", body, time.Now())

	result, err := New(Config{}).VerifyWebhook(providers.WebhookVerificationRequest{
		Headers:       map[string]string{"Stripe-Signature": signature},
		Body:          body,
		WebhookSecret: "whsec_test",
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() = %v", err)
	}
	if result.Type != constants.EventPaymentFailed || result.Status != string(constants.PaymentFailed) {
		t.Fatalf("VerifyWebhook() = type %s status %s, want failed/failed", result.Type, result.Status)
	}
}

func TestVerifyWebhookMapsRefundToMerchantReference(t *testing.T) {
	body := []byte(`{"id":"evt_2","type":"charge.refunded","data":{"object":{"id":"ch_1","payment_intent":"pi_123","amount":2500,"amount_refunded":1000,"currency":"usd","metadata":{"reference":"pay-ref-1"}}}}`)
	signature := sign("whsec_test", body, time.Now())

	result, err := New(Config{}).VerifyWebhook(providers.WebhookVerificationRequest{
		Headers:       map[string]string{"Stripe-Signature": signature},
		Body:          body,
		WebhookSecret: "whsec_test",
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() = %v", err)
	}
	if result.Type != constants.EventRefundSucceeded || result.Reference != "pay-ref-1" ||
		result.ProviderReference != "pi_123" || result.AmountMinor != 1000 {
		t.Fatalf("VerifyWebhook() = %#v", result)
	}
}

// sign builds the Stripe-Signature header the way Stripe does.
func sign(secret string, body []byte, at time.Time) string {
	timestamp := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return fmt.Sprintf("t=%s,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}
