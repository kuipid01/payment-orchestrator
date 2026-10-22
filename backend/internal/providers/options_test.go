package providers

import (
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

func boolPtr(value bool) *bool { return &value }

func TestPaymentOptionsRejectsTwoNamespaces(t *testing.T) {
	options := PaymentOptions{
		Stripe: &StripeOptions{ProductName: "Pro plan"},
		PayPal: &PayPalOptions{Description: "Pro plan"},
	}
	if err := options.Validate(); !apperrors.Is(err, codeOptionsConflict) {
		t.Fatalf("Validate() = %v, want %s", err, codeOptionsConflict)
	}
}

func TestStripeStandardPaymentsNeedsNoTaxCode(t *testing.T) {
	options := PaymentOptions{Stripe: &StripeOptions{ProductName: "Pro plan"}}
	if err := options.ValidateFor(constants.ProviderStripe, false); err != nil {
		t.Fatalf("standard options rejected: %v", err)
	}
}

func TestStripeManagedPaymentsRequiresEligibleTaxCode(t *testing.T) {
	options := PaymentOptions{Stripe: &StripeOptions{MerchantOfRecord: "managed"}}
	err := options.ValidateFor(constants.ProviderStripe, false)
	if !apperrors.Is(err, codeOptionsInvalid) {
		t.Fatalf("ValidateFor() = %v, want %s", err, codeOptionsInvalid)
	}

	// A fabricated-looking code is rejected before it reaches Stripe.
	fabricated := PaymentOptions{Stripe: &StripeOptions{
		MerchantOfRecord: "managed",
		ProductTaxCode:   "digital-goods",
	}}
	if err := fabricated.ValidateFor(constants.ProviderStripe, false); !apperrors.Is(err, codeOptionsInvalid) {
		t.Fatalf("fabricated tax code accepted: %v", err)
	}

	eligible := PaymentOptions{Stripe: &StripeOptions{
		MerchantOfRecord: "managed",
		ProductTaxCode:   "txcd_10000000",
	}}
	if err := eligible.ValidateFor(constants.ProviderStripe, false); err != nil {
		t.Fatalf("eligible managed options rejected: %v", err)
	}
}

func TestStripeManagedPaymentsRejectedForTestTransaction(t *testing.T) {
	options := PaymentOptions{Stripe: &StripeOptions{
		MerchantOfRecord: "managed",
		ProductTaxCode:   "txcd_10000000",
	}}
	if err := options.ValidateFor(constants.ProviderStripe, true); !apperrors.Is(err, codeOptionsTestUnsupported) {
		t.Fatalf("ValidateFor(test) = %v, want %s", err, codeOptionsTestUnsupported)
	}
}

func TestStripeOptionsRejectInsecureReturnURL(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "/relative/path", "ftp://store.example/return"} {
		options := PaymentOptions{Stripe: &StripeOptions{SuccessURL: raw}}
		if err := options.Validate(); !apperrors.Is(err, codeOptionsInvalid) {
			t.Fatalf("success_url %q accepted: %v", raw, err)
		}
	}
	options := PaymentOptions{Stripe: &StripeOptions{SuccessURL: "https://store.example/return"}}
	if err := options.Validate(); err != nil {
		t.Fatalf("valid success_url rejected: %v", err)
	}
}

func TestStripeOptionsRejectUnsupportedCheckoutValues(t *testing.T) {
	cases := []*StripeOptions{
		{SubmitType: "subscribe"},
		{BillingAddressCollection: "sometimes"},
		{SetupFutureUsage: "forever"},
		{CustomerEmail: "not-an-email"},
		{PaymentIntentMetadata: map[string]string{"": "value"}},
	}
	for _, options := range cases {
		if err := (PaymentOptions{Stripe: options}).Validate(); err == nil {
			t.Fatalf("options %#v accepted", options)
		}
	}
}

func TestPayPalOptionsNormalizeAndValidate(t *testing.T) {
	options := PaymentOptions{PayPal: &PayPalOptions{
		Description:        "Pro plan",
		InvoiceID:          "INV-42",
		CustomID:           "order_42",
		PayerEmail:         "payer@example.com",
		BrandName:          "Acme",
		Locale:             "en-US",
		LandingPage:        "login",
		UserAction:         "pay_now",
		ShippingPreference: "no_shipping",
	}}
	if err := options.ValidateFor(constants.ProviderPayPal, false); err != nil {
		t.Fatalf("valid PayPal options rejected: %v", err)
	}
	if options.PayPal.LandingPage != "LOGIN" || options.PayPal.UserAction != "PAY_NOW" ||
		options.PayPal.ShippingPreference != "NO_SHIPPING" {
		t.Fatalf("PayPal enums not normalized: %#v", options.PayPal)
	}

	invalid := PaymentOptions{PayPal: &PayPalOptions{LandingPage: "SOMEWHERE"}}
	if err := invalid.Validate(); !apperrors.Is(err, codeOptionsInvalid) {
		t.Fatalf("landing_page accepted: %v", err)
	}
}

func TestValidateForRejectsForeignProvider(t *testing.T) {
	options := PaymentOptions{Stripe: &StripeOptions{ProductName: "Pro plan"}}
	if err := options.ValidateFor(constants.ProviderPayPal, false); !apperrors.Is(err, codeOptionsUnsupported) {
		t.Fatalf("ValidateFor(paypal) = %v, want %s", err, codeOptionsUnsupported)
	}
}

func TestOptionKeysAdvertiseOnlySupportedProviders(t *testing.T) {
	if len(OptionKeys(constants.ProviderStripe)) == 0 || len(OptionKeys(constants.ProviderPayPal)) == 0 {
		t.Fatal("stripe and paypal must advertise their supported options")
	}
	if got := OptionKeys(constants.ProviderPaystack); len(got) != 0 {
		t.Fatalf("paystack advertises options: %v", got)
	}
}
