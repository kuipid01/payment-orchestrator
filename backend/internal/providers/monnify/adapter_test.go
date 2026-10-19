package monnify

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

func testCredentials() providers.Credentials {
	return providers.Credentials{
		APIKey:       "monnify-test-api-key",
		SecretKey:    "monnify-test-secret-key",
		ContractCode: "1234567890",
		Environment:  constants.EnvironmentTest,
	}
}

func TestTestCredentialsUsesSandboxBasicAuth(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("path = %q, want Monnify auth endpoint", r.URL.Path)
		}
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte("monnify-test-api-key:monnify-test-secret-key"))
		if got := r.Header.Get("Authorization"); got != want {
			t.Errorf("Authorization = %q, want Basic credentials", got)
		}
		w.Write([]byte(`{"requestSuccessful":true,"responseBody":{"accessToken":"test-token"}}`))
	})

	if err := adapter.TestCredentials(context.Background(), testCredentials()); err != nil {
		t.Fatalf("TestCredentials() = %v", err)
	}
}

func TestCreatePaymentInitializesHostedCheckout(t *testing.T) {
	calls := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Write([]byte(`{"requestSuccessful":true,"responseBody":{"accessToken":"test-token"}}`))
			return
		}
		if r.URL.Path != "/api/v1/merchant/transactions/init-transaction" {
			t.Errorf("path = %q, want transaction initialization endpoint", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer access token", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["amount"] != float64(125.5) || body["contractCode"] != "1234567890" ||
			body["paymentReference"] != "pay-ref-1" || body["currencyCode"] != "NGN" {
			t.Errorf("transaction payload = %#v", body)
		}
		w.Write([]byte(`{"requestSuccessful":true,"responseBody":{"transactionReference":"TX-1","paymentReference":"pay-ref-1","checkoutUrl":"https://sandbox.monnify.com/checkout","paymentStatus":"PENDING","amount":125.5,"currency":"NGN"}}`))
	})

	result, err := adapter.CreatePayment(context.Background(), testCredentials(), providers.PaymentRequest{
		Reference: "pay-ref-1", AmountMinor: 12550, Currency: "NGN", Email: "payer@example.com",
	})
	if err != nil {
		t.Fatalf("CreatePayment() = %v", err)
	}
	if result.ProviderReference != "TX-1" || result.AuthorizationURL != "https://sandbox.monnify.com/checkout" {
		t.Fatalf("CreatePayment() result = %#v", result)
	}
	if result.AmountMinor != 12550 || result.Status != string(constants.PaymentProcessing) {
		t.Fatalf("CreatePayment() amount/status = %d/%s", result.AmountMinor, result.Status)
	}
}

func TestVerifyWebhookAcceptsProductionSignature(t *testing.T) {
	body := []byte(`{"eventType":"SUCCESSFUL_TRANSACTION","eventData":{"paymentReference":"pay-ref-1","transactionReference":"TX-1","paymentStatus":"PAID","amountPaid":125.5,"currency":"NGN"}}`)
	mac := hmac.New(sha512.New, []byte("webhook-secret"))
	_, _ = mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))
	adapter := New(Config{})

	result, err := adapter.VerifyWebhook(providers.WebhookVerificationRequest{
		Headers:     map[string]string{"monnify-signature": signature},
		Body:        body,
		SecretKey:   "webhook-secret",
		Environment: constants.EnvironmentLive,
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() = %v", err)
	}
	if !result.SignatureValid || result.Reference != "pay-ref-1" ||
		result.ProviderReference != "TX-1" ||
		result.Type != constants.EventPaymentSucceeded ||
		result.AmountMinor != 12550 {
		t.Fatalf("VerifyWebhook() result = %#v", result)
	}
}

func TestVerifyWebhookAllowsUnsignedSandboxNotification(t *testing.T) {
	adapter := New(Config{})
	result, err := adapter.VerifyWebhook(providers.WebhookVerificationRequest{
		Body:        []byte(`{"eventType":"SUCCESSFUL_TRANSACTION","eventData":{"paymentReference":"pay-ref-1","paymentStatus":"PAID"}}`),
		Environment: constants.EnvironmentTest,
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() = %v", err)
	}
	if !result.SignatureValid {
		t.Fatal("sandbox notification was not accepted")
	}
}

func TestVerifyWebhookRejectsUnsignedProductionNotification(t *testing.T) {
	adapter := New(Config{})
	_, err := adapter.VerifyWebhook(providers.WebhookVerificationRequest{
		Body:        []byte(`{"eventType":"SUCCESSFUL_TRANSACTION","eventData":{}}`),
		Environment: constants.EnvironmentLive,
	})
	if err == nil {
		t.Fatal("VerifyWebhook() accepted unsigned production event")
	}
}

func newTestAdapter(t *testing.T, handler http.HandlerFunc) *Adapter {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return New(Config{TestBaseURL: server.URL, LiveBaseURL: server.URL})
}

func TestVerifyWebhookAcceptsOfficialSecretPrefixedSHA512Signature(t *testing.T) {
	body := []byte(`{"eventType":"SUCCESSFUL_TRANSACTION","eventData":{"paymentReference":"pay-ref-1","paymentStatus":"PAID"}}`)
	digestInput := append([]byte("webhook-secret"), body...)
	digest := sha512.Sum512(digestInput)
	signature := hex.EncodeToString(digest[:])
	adapter := New(Config{})

	result, err := adapter.VerifyWebhook(providers.WebhookVerificationRequest{
		Headers:     map[string]string{"monnify-signature": signature},
		Body:        body,
		SecretKey:   "webhook-secret",
		Environment: constants.EnvironmentLive,
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() = %v", err)
	}
	if !result.SignatureValid {
		t.Fatal("signature was not marked valid")
	}
}

func TestDefaultBaseURLsMatchMonnifyEnvironments(t *testing.T) {
	adapter := New(Config{})
	if got := adapter.baseURL(constants.EnvironmentTest); got != "https://sandbox.monnify.com" {
		t.Errorf("test base URL = %q", got)
	}
	if got := adapter.baseURL(constants.EnvironmentLive); got != "https://api.monnify.com" {
		t.Errorf("live base URL = %q", got)
	}
}
