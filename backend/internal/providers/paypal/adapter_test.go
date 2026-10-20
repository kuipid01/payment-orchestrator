package paypal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

func newTestAdapter(t *testing.T, handler http.HandlerFunc) *Adapter {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return New(Config{TestBaseURL: server.URL, LiveBaseURL: server.URL})
}

func testCreds() providers.Credentials {
	return providers.Credentials{
		PublicKey:   "paypal-client-id",
		SecretKey:   "paypal-client-secret",
		Environment: constants.EnvironmentTest,
	}
}

const tokenBody = `{"access_token":"A21AA-token","expires_in":32400,"token_type":"Bearer"}`

// paypalStub routes a request to the handler for its path, always answering the
// OAuth token call first.
func paypalStub(t *testing.T, handler http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth2/token" {
			if r.Method != http.MethodPost {
				t.Errorf("token method = %s", r.Method)
			}
			want := "Basic " + base64.StdEncoding.EncodeToString([]byte("paypal-client-id:paypal-client-secret"))
			if got := r.Header.Get("Authorization"); got != want {
				t.Errorf("token Authorization = %q", got)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if got := r.PostForm.Get("grant_type"); got != "client_credentials" {
				t.Errorf("grant_type = %q", got)
			}
			w.Write([]byte(tokenBody))
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer A21AA-token" {
			t.Errorf("Authorization = %q", got)
		}
		handler(w, r)
	}
}

func TestTestCredentialsMintsPayPalToken(t *testing.T) {
	adapter := newTestAdapter(t, paypalStub(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected call %s", r.URL.Path)
	}))
	if err := adapter.TestCredentials(context.Background(), testCreds()); err != nil {
		t.Fatalf("TestCredentials() = %v", err)
	}
}

func TestTokenIsCachedAcrossCalls(t *testing.T) {
	tokenCalls := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/oauth2/token" {
			t.Errorf("unexpected call %s", r.URL.Path)
			return
		}
		tokenCalls++
		w.Write([]byte(tokenBody))
	})
	for i := 0; i < 2; i++ {
		if err := adapter.TestCredentials(context.Background(), testCreds()); err != nil {
			t.Fatalf("TestCredentials() = %v", err)
		}
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d, want 1 (cached)", tokenCalls)
	}
}

func TestCreatePaymentCreatesOrderWithApprovalLink(t *testing.T) {
	adapter := newTestAdapter(t, paypalStub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/checkout/orders" || r.Method != http.MethodPost {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("PayPal-Request-Id"); got != "order:pay-ref-1" {
			t.Errorf("PayPal-Request-Id = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["intent"] != "CAPTURE" {
			t.Errorf("intent = %v", body["intent"])
		}
		units := body["purchase_units"].([]any)
		unit := units[0].(map[string]any)
		if unit["custom_id"] != "pay-ref-1" {
			t.Errorf("custom_id = %v", unit["custom_id"])
		}
		amount := unit["amount"].(map[string]any)
		if amount["value"] != "25.00" || amount["currency_code"] != "USD" {
			t.Errorf("amount = %#v", amount)
		}
		w.Write([]byte(`{"id":"ORDER-1","status":"CREATED","purchase_units":[{"custom_id":"pay-ref-1","amount":{"currency_code":"USD","value":"25.00"}}],"links":[{"rel":"approve","href":"https://www.sandbox.paypal.com/checkoutnow?token=ORDER-1","method":"GET"}]}`))
	}))

	result, err := adapter.CreatePayment(context.Background(), testCreds(), providers.PaymentRequest{
		Reference:   "pay-ref-1",
		AmountMinor: 2500,
		Currency:    "USD",
		Email:       "payer@example.com",
		CallbackURL: "https://store.example/return",
	})
	if err != nil {
		t.Fatalf("CreatePayment() = %v", err)
	}
	if result.ProviderReference != "ORDER-1" {
		t.Errorf("ProviderReference = %q", result.ProviderReference)
	}
	if result.AuthorizationURL != "https://www.sandbox.paypal.com/checkoutnow?token=ORDER-1" {
		t.Errorf("AuthorizationURL = %q", result.AuthorizationURL)
	}
	if result.Status != string(constants.PaymentPending) || result.AmountMinor != 2500 {
		t.Errorf("status/amount = %s/%d", result.Status, result.AmountMinor)
	}
}

func TestCreatePaymentRejectsUnsupportedCurrency(t *testing.T) {
	adapter := New(Config{})
	_, err := adapter.CreatePayment(context.Background(), testCreds(), providers.PaymentRequest{
		Reference:   "pay-ref-1",
		AmountMinor: 2500,
		Currency:    "NGN",
		CallbackURL: "https://store.example/return",
	})
	if !apperrors.Is(err, "paypal_currency_unsupported") {
		t.Fatalf("CreatePayment() error = %v, want currency rejection", err)
	}
}

func TestVerifyPaymentCapturesApprovedOrder(t *testing.T) {
	captured := false
	adapter := newTestAdapter(t, paypalStub(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/checkout/orders/ORDER-1":
			if captured {
				w.Write([]byte(`{"id":"ORDER-1","status":"COMPLETED","purchase_units":[{"custom_id":"pay-ref-1","amount":{"currency_code":"USD","value":"25.00"},"payments":{"captures":[{"id":"CAP-1","status":"COMPLETED","amount":{"currency_code":"USD","value":"25.00"}}]}}]}`))
				return
			}
			w.Write([]byte(`{"id":"ORDER-1","status":"APPROVED","purchase_units":[{"custom_id":"pay-ref-1","amount":{"currency_code":"USD","value":"25.00"}}]}`))
		case "/v2/checkout/orders/ORDER-1/capture":
			captured = true
			if got := r.Header.Get("PayPal-Request-Id"); got != "capture:ORDER-1" {
				t.Errorf("capture request id = %q", got)
			}
			w.Write([]byte(`{"id":"ORDER-1","status":"COMPLETED","purchase_units":[{"custom_id":"pay-ref-1","amount":{"currency_code":"USD","value":"25.00"},"payments":{"captures":[{"id":"CAP-1","status":"COMPLETED","amount":{"currency_code":"USD","value":"25.00"}}]}}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))

	result, err := adapter.VerifyPayment(context.Background(), testCreds(), "ORDER-1")
	if err != nil {
		t.Fatalf("VerifyPayment() = %v", err)
	}
	if !captured {
		t.Fatal("approved order was not captured")
	}
	if result.Status != string(constants.PaymentSuccessful) || result.AmountMinor != 2500 {
		t.Fatalf("VerifyPayment() = %#v", result)
	}
	if result.ProviderReference != "ORDER-1" {
		t.Errorf("ProviderReference = %q, want the order id", result.ProviderReference)
	}
}

func TestRefundPaymentResolvesCaptureFromOrder(t *testing.T) {
	adapter := newTestAdapter(t, paypalStub(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/checkout/orders/ORDER-1":
			w.Write([]byte(`{"id":"ORDER-1","status":"COMPLETED","purchase_units":[{"amount":{"currency_code":"USD","value":"25.00"},"payments":{"captures":[{"id":"CAP-1","status":"COMPLETED","amount":{"currency_code":"USD","value":"25.00"}}]}}]}`))
		case "/v2/payments/captures/CAP-1/refunds":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			amount := body["amount"].(map[string]any)
			if amount["value"] != "10.00" || amount["currency_code"] != "USD" {
				t.Errorf("refund amount = %#v", amount)
			}
			w.Write([]byte(`{"id":"REF-1","status":"COMPLETED","amount":{"currency_code":"USD","value":"10.00"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))

	result, err := adapter.RefundPayment(context.Background(), testCreds(), providers.RefundRequest{
		PaymentReference:  "pay-ref-1",
		ProviderReference: "ORDER-1",
		AmountMinor:       1000,
		Currency:          "USD",
	})
	if err != nil {
		t.Fatalf("RefundPayment() = %v", err)
	}
	if result.ProviderRefundID != "REF-1" || result.Status != string(constants.RefundSuccessful) {
		t.Fatalf("RefundPayment() = %#v", result)
	}
	if result.AmountMinor != 1000 {
		t.Errorf("AmountMinor = %d", result.AmountMinor)
	}
}

func TestVerifyWebhookRequiresPayPalApproval(t *testing.T) {
	body := []byte(`{"id":"WH-1","event_type":"PAYMENT.CAPTURE.COMPLETED","resource":{"id":"CAP-1","status":"COMPLETED","custom_id":"pay-ref-1","amount":{"currency_code":"USD","value":"25.00"},"supplementary_data":{"related_ids":{"order_id":"ORDER-1"}}}}`)
	adapter := newTestAdapter(t, paypalStub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/notifications/verify-webhook-signature" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["webhook_id"] != "WH-REG-1" || payload["transmission_id"] != "TX-1" {
			t.Errorf("verification payload = %#v", payload)
		}
		w.Write([]byte(`{"verification_status":"SUCCESS"}`))
	}))

	result, err := adapter.VerifyWebhook(providers.WebhookVerificationRequest{
		Headers: map[string]string{
			"paypal-transmission-id":   "TX-1",
			"paypal-transmission-time": "2026-10-20T10:00:00Z",
			"paypal-cert-url":          "https://api.paypal.com/cert.pem",
			"paypal-auth-algo":         "SHA256withRSA",
			"paypal-transmission-sig":  "sig",
		},
		Body:          body,
		PublicKey:     "paypal-client-id",
		SecretKey:     "paypal-client-secret",
		WebhookSecret: "WH-REG-1",
		Environment:   constants.EnvironmentTest,
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() = %v", err)
	}
	if !result.SignatureValid || result.Type != constants.EventPaymentSucceeded {
		t.Fatalf("VerifyWebhook() = %#v", result)
	}
	if result.ProviderReference != "ORDER-1" || result.Reference != "pay-ref-1" {
		t.Errorf("refs = %q/%q", result.ProviderReference, result.Reference)
	}
	if result.AmountMinor != 2500 || result.Status != string(constants.PaymentSuccessful) {
		t.Errorf("amount/status = %d/%s", result.AmountMinor, result.Status)
	}
}

func TestVerifyWebhookRejectsFailedVerification(t *testing.T) {
	body := []byte(`{"id":"WH-1","event_type":"PAYMENT.CAPTURE.COMPLETED","resource":{"id":"CAP-1","amount":{"currency_code":"USD","value":"25.00"}}}`)
	adapter := newTestAdapter(t, paypalStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"verification_status":"FAILURE"}`))
	}))

	_, err := adapter.VerifyWebhook(providers.WebhookVerificationRequest{
		Headers: map[string]string{
			"paypal-transmission-id":   "TX-1",
			"paypal-transmission-time": "2026-10-20T10:00:00Z",
			"paypal-cert-url":          "https://api.paypal.com/cert.pem",
			"paypal-auth-algo":         "SHA256withRSA",
			"paypal-transmission-sig":  "sig",
		},
		Body:          body,
		PublicKey:     "paypal-client-id",
		SecretKey:     "paypal-client-secret",
		WebhookSecret: "WH-REG-1",
		Environment:   constants.EnvironmentTest,
	})
	if !apperrors.Is(err, apperrors.CodeWebhookSignatureInvalid) {
		t.Fatalf("VerifyWebhook() error = %v, want signature rejection", err)
	}
}

func TestVerifyWebhookCapturesApprovedOrder(t *testing.T) {
	body := []byte(`{"id":"WH-2","event_type":"CHECKOUT.ORDER.APPROVED","resource":{"id":"ORDER-1","status":"APPROVED","purchase_units":[{"custom_id":"pay-ref-1","amount":{"currency_code":"USD","value":"25.00"}}]}}`)
	adapter := newTestAdapter(t, paypalStub(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/notifications/verify-webhook-signature":
			w.Write([]byte(`{"verification_status":"SUCCESS"}`))
		case "/v2/checkout/orders/ORDER-1/capture":
			w.Write([]byte(`{"id":"ORDER-1","status":"COMPLETED","purchase_units":[{"custom_id":"pay-ref-1","amount":{"currency_code":"USD","value":"25.00"},"payments":{"captures":[{"id":"CAP-1","status":"COMPLETED","amount":{"currency_code":"USD","value":"25.00"}}]}}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))

	result, err := adapter.VerifyWebhook(providers.WebhookVerificationRequest{
		Headers: map[string]string{
			"paypal-transmission-id":   "TX-2",
			"paypal-transmission-time": "2026-10-20T10:00:00Z",
			"paypal-cert-url":          "https://api.paypal.com/cert.pem",
			"paypal-auth-algo":         "SHA256withRSA",
			"paypal-transmission-sig":  "sig",
		},
		Body:          body,
		PublicKey:     "paypal-client-id",
		SecretKey:     "paypal-client-secret",
		WebhookSecret: "WH-REG-1",
		Environment:   constants.EnvironmentTest,
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() = %v", err)
	}
	if result.Type != constants.EventPaymentSucceeded || result.Status != string(constants.PaymentSuccessful) {
		t.Fatalf("VerifyWebhook() = %#v, want an approval captured and confirmed", result)
	}
	if result.AmountMinor != 2500 || result.Currency != "USD" {
		t.Errorf("amount/currency = %d/%s", result.AmountMinor, result.Currency)
	}
}

func TestVerifyWebhookRejectsMissingRegistration(t *testing.T) {
	adapter := New(Config{})
	_, err := adapter.VerifyWebhook(providers.WebhookVerificationRequest{
		Body: []byte(`{}`),
	})
	if !apperrors.Is(err, apperrors.CodeWebhookSignatureInvalid) {
		t.Fatalf("VerifyWebhook() error = %v, want signature rejection", err)
	}
}

func TestDefaultBaseURLsMatchPayPalEnvironments(t *testing.T) {
	adapter := New(Config{})
	if got := adapter.baseURL(constants.EnvironmentTest); got != "https://api-m.sandbox.paypal.com" {
		t.Errorf("test base URL = %q", got)
	}
	if got := adapter.baseURL(constants.EnvironmentLive); !strings.HasPrefix(got, "https://api-m.paypal.com") {
		t.Errorf("live base URL = %q", got)
	}
}
