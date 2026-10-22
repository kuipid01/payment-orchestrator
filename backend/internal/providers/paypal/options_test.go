package paypal

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kuipid01/payconnect/internal/providers"
)

const orderBody = `{"id":"ORDER-1","status":"CREATED","links":[{"rel":"approve","href":"https://www.sandbox.paypal.com/checkoutnow?token=ORDER-1","method":"GET"}]}`

// captureOrderBody runs one CreatePayment and returns the decoded order body.
func captureOrderBody(t *testing.T, req providers.PaymentRequest) map[string]any {
	t.Helper()
	var body map[string]any
	adapter := newTestAdapter(t, paypalStub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/checkout/orders" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(orderBody))
	}))
	if _, err := adapter.CreatePayment(context.Background(), testCreds(), req); err != nil {
		t.Fatalf("CreatePayment() = %v", err)
	}
	return body
}

func baseOrderRequest() providers.PaymentRequest {
	return providers.PaymentRequest{
		Reference:   "pay-ref-1",
		AmountMinor: 2500,
		Currency:    "USD",
		CallbackURL: "https://store.example/return",
	}
}

func TestOrderOptionsAreForwardedToPayPal(t *testing.T) {
	req := baseOrderRequest()
	req.Options = providers.PaymentOptions{PayPal: &providers.PayPalOptions{
		Description:        "Pro plan",
		InvoiceID:          "INV-42",
		CustomID:           "order_42",
		PayerEmail:         "payer@example.com",
		BrandName:          "Acme Store",
		Locale:             "en-US",
		LandingPage:        "LOGIN",
		UserAction:         "PAY_NOW",
		ShippingPreference: "NO_SHIPPING",
	}}
	body := captureOrderBody(t, req)

	unit := body["purchase_units"].([]any)[0].(map[string]any)
	if unit["description"] != "Pro plan" {
		t.Errorf("description = %v", unit["description"])
	}
	if unit["invoice_id"] != "INV-42" {
		t.Errorf("invoice_id = %v", unit["invoice_id"])
	}
	if unit["custom_id"] != "order_42" {
		t.Errorf("custom_id = %v", unit["custom_id"])
	}
	// reference_id stays PayConnect's own identifier so reconciliation is stable.
	if unit["reference_id"] != "pay-ref-1" {
		t.Errorf("reference_id = %v", unit["reference_id"])
	}

	context := body["application_context"].(map[string]any)
	for field, want := range map[string]any{
		"brand_name":          "Acme Store",
		"locale":              "en-US",
		"landing_page":        "LOGIN",
		"user_action":         "PAY_NOW",
		"shipping_preference": "NO_SHIPPING",
	} {
		if got := context[field]; got != want {
			t.Errorf("application_context[%s] = %v, want %v", field, got, want)
		}
	}

	payer := body["payer"].(map[string]any)
	if payer["email_address"] != "payer@example.com" {
		t.Errorf("payer.email_address = %v", payer["email_address"])
	}
}

func TestDefaultsKeepPayConnectIdentifiers(t *testing.T) {
	body := captureOrderBody(t, baseOrderRequest())

	unit := body["purchase_units"].([]any)[0].(map[string]any)
	if unit["custom_id"] != "pay-ref-1" || unit["invoice_id"] != "pay-ref-1" {
		t.Errorf("default identifiers = %v/%v", unit["custom_id"], unit["invoice_id"])
	}
	if unit["description"] != "Payment pay-ref-1" {
		t.Errorf("default description = %v", unit["description"])
	}
	context := body["application_context"].(map[string]any)
	if context["user_action"] != "PAY_NOW" || context["shipping_preference"] != "NO_SHIPPING" {
		t.Errorf("defaults = %v", context)
	}
	if _, ok := body["payer"]; ok {
		t.Errorf("payer sent without an email: %v", body["payer"])
	}
}

func TestTestTransactionLabelsItself(t *testing.T) {
	req := baseOrderRequest()
	req.Test = true
	body := captureOrderBody(t, req)

	unit := body["purchase_units"].([]any)[0].(map[string]any)
	if unit["description"] != "Test payment pay-ref-1" {
		t.Errorf("test description = %v", unit["description"])
	}
}
