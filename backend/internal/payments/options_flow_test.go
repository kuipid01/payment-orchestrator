package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
	"github.com/kuipid01/payconnect/internal/providers/stripe"
)

// TestProviderOptionsFlowFromJSONToStripeRequest walks the real chain for one
// request: the JSON body a client posts → the shared payment contract → routing
// validation → the adapter's provider request. It is the contract the dashboard
// and the Test Arena speak, so a mismatch there is caught here.
func TestProviderOptionsFlowFromJSONToStripeRequest(t *testing.T) {
	body := `{
		"amount": 20000,
		"currency": "USD",
		"customer": { "email": "customer@example.com" },
		"callback_url": "https://store.example/return",
		"options": {
			"stripe": {
				"merchant_of_record": "managed",
				"product_name": "Digital guide",
				"product_description": "PDF download",
				"product_tax_code": "txcd_10000000",
				"payment_intent_metadata": { "order_id": "1042" }
			}
		}
	}`
	var request CreateRequest
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		t.Fatalf("decode request: %v", err)
	}

	normalized, amount, currency, reference, err := normalizeCreate(CreateInput{
		MerchantID:  "mer_1",
		Environment: constants.EnvironmentTest,
		Request:     request,
	})
	if err != nil {
		t.Fatalf("normalizeCreate() = %v", err)
	}
	if normalized.Adapter != constants.ProviderStripe.String() {
		t.Fatalf("adapter = %q, want stripe", normalized.Adapter)
	}

	var captured url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		captured = r.PostForm
		w.Write([]byte(`{"id":"cs_test_1","status":"open","payment_status":"unpaid","amount_total":20000,"currency":"usd","url":"https://checkout.stripe.com/c/pay/cs_test_1"}`))
	}))
	defer server.Close()

	adapter := stripe.New(stripe.Config{BaseURL: server.URL})
	_, err = adapter.CreatePayment(context.Background(),
		providers.Credentials{SecretKey: "sk_test_1234567890abcdef", Environment: constants.EnvironmentTest},
		providers.PaymentRequest{
			Reference:   reference,
			AmountMinor: amount,
			Currency:    currency,
			Email:       normalized.Customer.Email,
			Description: normalized.Description,
			CallbackURL: normalized.CallbackURL,
			Metadata:    normalized.Metadata,
			Options:     normalized.Options,
		})
	if err != nil {
		t.Fatalf("CreatePayment() = %v", err)
	}

	expect := map[string]string{
		"managed_payments[enabled]":                         "true",
		"line_items[0][price_data][product_data][name]":     "Digital guide",
		"line_items[0][price_data][product_data][description]": "PDF download",
		"line_items[0][price_data][product_data][tax_code]": "txcd_10000000",
		"payment_intent_data[metadata][order_id]":           "1042",
	}
	for field, want := range expect {
		if got := captured.Get(field); got != want {
			t.Errorf("%s = %q, want %q", field, got, want)
		}
	}
}

func TestUnsupportedOptionsAreRejectedBeforeRouting(t *testing.T) {
	bodies := map[string]string{
		// A provider PayConnect does not support at all.
		"unknown namespace": `{"amount":100,"currency":"USD","customer":{"email":"payer@example.com"},"options":{"paystack":{"fee":"absorb"}}}`,
		// A real provider namespace with a field the contract does not define.
		"unknown field": `{"amount":100,"currency":"USD","customer":{"email":"payer@example.com"},"options":{"stripe":{"bogus":"value"}}}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			var request CreateRequest
			if err := json.Unmarshal([]byte(body), &request); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			_, _, _, _, err := normalizeCreate(CreateInput{
				MerchantID:  "mer_1",
				Environment: constants.EnvironmentTest,
				Request:     request,
			})
			if !apperrors.Is(err, "provider_options_not_supported") {
				t.Fatalf("normalizeCreate() = %v, want provider_options_not_supported", err)
			}
		})
	}
}
