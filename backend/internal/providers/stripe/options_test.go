package stripe

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

const checkoutSessionBody = `{"id":"cs_test_123","object":"checkout.session","status":"open","payment_status":"unpaid","amount_total":2500,"currency":"usd","url":"https://checkout.stripe.com/c/pay/cs_test_123","client_reference_id":"pay-ref-1"}`

// captureCheckoutForm runs one CreatePayment against a stub Stripe that records
// the form it was sent.
func captureCheckoutForm(t *testing.T, req providers.PaymentRequest) url.Values {
	t.Helper()
	var captured url.Values
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		captured = r.PostForm
		w.Write([]byte(checkoutSessionBody))
	})
	if _, err := adapter.CreatePayment(context.Background(), testCreds(), req); err != nil {
		t.Fatalf("CreatePayment() = %v", err)
	}
	return captured
}

func baseRequest() providers.PaymentRequest {
	return providers.PaymentRequest{
		Reference:   "pay-ref-1",
		AmountMinor: 2500,
		Currency:    "USD",
		CallbackURL: "https://store.example/return",
	}
}

func TestStandardCheckoutSendsNoManagedPaymentsConfiguration(t *testing.T) {
	req := baseRequest()
	req.Options = providers.PaymentOptions{Stripe: &providers.StripeOptions{
		ProductName:        "Pro plan",
		ProductDescription: "Annual subscription",
	}}
	form := captureCheckoutForm(t, req)

	if got := form.Get("managed_payments[enabled]"); got != "" {
		t.Errorf("managed_payments[enabled] = %q, want unset for standard payments", got)
	}
	if got := form.Get("line_items[0][price_data][product_data][tax_code]"); got != "" {
		t.Errorf("tax_code = %q, want unset when the caller supplied none", got)
	}
	if got := form.Get("line_items[0][price_data][product_data][name]"); got != "Pro plan" {
		t.Errorf("product name = %q", got)
	}
	if got := form.Get("line_items[0][price_data][product_data][description]"); got != "Annual subscription" {
		t.Errorf("product description = %q", got)
	}
}

func TestManagedPaymentsSendsEnabledAndTaxCode(t *testing.T) {
	req := baseRequest()
	req.Options = providers.PaymentOptions{Stripe: &providers.StripeOptions{
		MerchantOfRecord: "managed",
		ProductName:      "Digital guide",
		ProductTaxCode:   "txcd_10000000",
	}}
	form := captureCheckoutForm(t, req)

	if got := form.Get("managed_payments[enabled]"); got != "true" {
		t.Errorf("managed_payments[enabled] = %q, want true", got)
	}
	if got := form.Get("line_items[0][price_data][product_data][tax_code]"); got != "txcd_10000000" {
		t.Errorf("tax_code = %q", got)
	}
}

func TestTestTransactionUsesStandardPaymentsAndGeneratedName(t *testing.T) {
	req := baseRequest()
	req.Test = true
	form := captureCheckoutForm(t, req)

	if got := form.Get("managed_payments[enabled]"); got != "" {
		t.Errorf("test transaction enabled managed payments: %q", got)
	}
	if got := form.Get("line_items[0][price_data][product_data][name]"); got != "Test payment pay-ref-1" {
		t.Errorf("generated test product name = %q", got)
	}
}

func TestTestTransactionRejectsManagedPayments(t *testing.T) {
	req := baseRequest()
	req.Test = true
	req.Options = providers.PaymentOptions{Stripe: &providers.StripeOptions{
		MerchantOfRecord: "managed",
		ProductTaxCode:   "txcd_10000000",
	}}
	adapter := New(Config{})
	_, err := adapter.CreatePayment(context.Background(), testCreds(), req)
	if !apperrors.Is(err, "provider_options_not_available_for_test") {
		t.Fatalf("CreatePayment() = %v, want a test-policy rejection", err)
	}
}

func TestExplicitReturnURLsOverrideCallback(t *testing.T) {
	req := baseRequest()
	req.CallbackURL = ""
	req.Options = providers.PaymentOptions{Stripe: &providers.StripeOptions{
		SuccessURL: "https://store.example/paid",
		CancelURL:  "https://store.example/cancelled",
	}}
	form := captureCheckoutForm(t, req)

	if got := form.Get("success_url"); got != "https://store.example/paid" {
		t.Errorf("success_url = %q", got)
	}
	if got := form.Get("cancel_url"); got != "https://store.example/cancelled" {
		t.Errorf("cancel_url = %q", got)
	}
}

func TestCheckoutAndPaymentIntentOptionsAreForwarded(t *testing.T) {
	req := baseRequest()
	allow := true
	req.Options = providers.PaymentOptions{Stripe: &providers.StripeOptions{
		CustomerEmail:            "override@example.com",
		SubmitType:               "pay",
		Locale:                   "en",
		BillingAddressCollection: "required",
		AllowPromotionCodes:      &allow,
		SetupFutureUsage:         "off_session",
		PaymentIntentMetadata:    map[string]string{"order_id": "42"},
	}}
	form := captureCheckoutForm(t, req)

	expect := map[string]string{
		"customer_email":             "override@example.com",
		"submit_type":                "pay",
		"locale":                     "en",
		"billing_address_collection": "required",
		"allow_promotion_codes":      "true",
		"payment_intent_data[setup_future_usage]":  "off_session",
		"payment_intent_data[metadata][order_id]":  "42",
		"payment_intent_data[metadata][reference]": "pay-ref-1",
	}
	for field, want := range expect {
		if got := form.Get(field); got != want {
			t.Errorf("%s = %q, want %q", field, got, want)
		}
	}
}
