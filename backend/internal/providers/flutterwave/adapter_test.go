package flutterwave

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

// newTestAdapter points the adapter at a stub server so these tests never touch
// the network.
func newTestAdapter(t *testing.T, handler http.HandlerFunc) *Adapter {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return New(Config{BaseURL: server.URL})
}

func testCreds() providers.Credentials {
	return providers.Credentials{SecretKey: "FLWSECK_TEST-stub", Environment: "test"}
}

// TestTestCredentialsAcceptsArrayData guards the regression that made connecting
// Flutterwave impossible: GET /balances returns `data` as an array, so decoding
// it into a map failed and a valid key was reported as "provider unavailable".
func TestTestCredentialsAcceptsArrayData(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/balances" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer FLWSECK_TEST-stub" {
			t.Errorf("authorization = %q", got)
		}
		// Shape taken from the live API: data is a list of balances.
		w.Write([]byte(`{"status":"success","message":"Wallet balances fetched","data":[{"currency":"NGN","available_balance":0}]}`))
	})

	if err := adapter.TestCredentials(context.Background(), testCreds()); err != nil {
		t.Fatalf("TestCredentials() = %v, want nil", err)
	}
}

// TestTestCredentialsRejectsBadKey ensures a rejected key is still reported as an
// authentication failure rather than a generic provider outage.
func TestTestCredentialsRejectsBadKey(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status":"error","message":"Invalid authorization key","data":null}`))
	})

	err := adapter.TestCredentials(context.Background(), testCreds())
	if !apperrors.Is(err, apperrors.CodeProviderAuthenticationFails) {
		t.Fatalf("TestCredentials() = %v, want provider authentication failure", err)
	}
}

// TestTestCredentialsReportsNonSuccessStatus covers the success-false body that
// Flutterwave returns for a rejected key with HTTP 200.
func TestTestCredentialsReportsNonSuccessStatus(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"error","message":"Invalid authorization key","data":[]}`))
	})

	err := adapter.TestCredentials(context.Background(), testCreds())
	if !apperrors.Is(err, apperrors.CodeProviderAuthenticationFails) {
		t.Fatalf("TestCredentials() = %v, want provider authentication failure", err)
	}
}

// TestCreatePaymentReadsLinkFromObjectData confirms the object-shaped payloads
// (payments, refunds, verification) still decode after Data became raw.
func TestCreatePaymentReadsLinkFromObjectData(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write([]byte(`{"status":"success","message":"Hosted Link","data":{"link":"https://checkout.flutterwave.com/v3/hosted/pay/abc"}}`))
	})

	result, err := adapter.CreatePayment(context.Background(), testCreds(), providers.PaymentRequest{
		Reference:   "ref-1",
		AmountMinor: 5000,
		Currency:    "NGN",
		Email:       "payer@example.com",
	})
	if err != nil {
		t.Fatalf("CreatePayment() = %v", err)
	}
	if result.AuthorizationURL != "https://checkout.flutterwave.com/v3/hosted/pay/abc" {
		t.Fatalf("AuthorizationURL = %q", result.AuthorizationURL)
	}
	if result.Status != string(constants.PaymentPending) {
		t.Fatalf("Status = %q", result.Status)
	}
}

// TestVerifyPaymentReadsTransactionObject confirms verification maps a
// successful transaction.
func TestVerifyPaymentReadsTransactionObject(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"success","message":"Transaction fetched","data":{"id":123,"tx_ref":"ref-1","status":"successful","amount":50,"charged_amount":50,"currency":"NGN","flw_ref":"FLW-1"}}`))
	})

	result, err := adapter.VerifyPayment(context.Background(), testCreds(), "ref-1")
	if err != nil {
		t.Fatalf("VerifyPayment() = %v", err)
	}
	if result.Status != string(constants.PaymentSuccessful) {
		t.Fatalf("Status = %q, want successful", result.Status)
	}
	if result.AmountMinor != 5000 {
		t.Fatalf("AmountMinor = %d, want 5000", result.AmountMinor)
	}
	if result.ProviderReference != "FLW-1" {
		t.Fatalf("ProviderReference = %q", result.ProviderReference)
	}
}

// TestRefundPaymentReadsObjectData covers the refund envelope, which also
// carries an object payload.
func TestRefundPaymentReadsObjectData(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/refunds" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write([]byte(`{"status":"success","message":"Refund created","data":{"id":987,"status":"completed"}}`))
	})

	result, err := adapter.RefundPayment(context.Background(), testCreds(), providers.RefundRequest{
		PaymentReference: "ref-1",
		AmountMinor:      5000,
		Currency:         "NGN",
	})
	if err != nil {
		t.Fatalf("RefundPayment() = %v", err)
	}
	if result.ProviderRefundID != "987" {
		t.Fatalf("ProviderRefundID = %q", result.ProviderRefundID)
	}
	if result.Status != string(constants.RefundSuccessful) {
		t.Fatalf("Status = %q", result.Status)
	}
}

// TestProviderUnavailableStillReportedForServerErrors keeps the catch-all honest:
// a provider outage must stay provider_unavailable.
func TestProviderUnavailableStillReportedForServerErrors(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status":"error","message":"boom"}`))
	})

	err := adapter.TestCredentials(context.Background(), testCreds())
	if !apperrors.Is(err, apperrors.CodeProviderUnavailable) {
		t.Fatalf("TestCredentials() = %v, want provider unavailable", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatal("unexpected context cancellation")
	}
}
