package payments

import (
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

func createInputWithOptions(options providers.PaymentOptions) CreateInput {
	return CreateInput{
		MerchantID:  "mer_1",
		Environment: constants.EnvironmentTest,
		Request: CreateRequest{
			Amount:      2500,
			Currency:    "USD",
			Customer:    CustomerInput{Email: "payer@example.com"},
			CallbackURL: "https://store.example/return",
			Options:     options,
		},
	}
}

func TestNormalizeCreatePinsAdapterFromOptions(t *testing.T) {
	in := createInputWithOptions(providers.PaymentOptions{Stripe: &providers.StripeOptions{
		ProductName: "Pro plan",
	}})
	req, _, _, _, err := normalizeCreate(in)
	if err != nil {
		t.Fatalf("normalizeCreate() = %v", err)
	}
	if req.Adapter != constants.ProviderStripe.String() {
		t.Fatalf("adapter = %q, want stripe", req.Adapter)
	}
}

func TestNormalizeCreateRejectsMismatchedAdapter(t *testing.T) {
	in := createInputWithOptions(providers.PaymentOptions{Stripe: &providers.StripeOptions{
		ProductName: "Pro plan",
	}})
	in.Request.Adapter = constants.ProviderPayPal.String()
	if _, _, _, _, err := normalizeCreate(in); !apperrors.Is(err, "provider_options_conflict") {
		t.Fatalf("normalizeCreate() = %v, want provider_options_conflict", err)
	}
}

func TestNormalizeCreateRejectsOptionsWithFallback(t *testing.T) {
	in := createInputWithOptions(providers.PaymentOptions{Stripe: &providers.StripeOptions{
		ProductName: "Pro plan",
	}})
	in.Request.Fallback = true
	if _, _, _, _, err := normalizeCreate(in); !apperrors.Is(err, "provider_options_not_supported") {
		t.Fatalf("normalizeCreate() = %v, want provider_options_not_supported", err)
	}
}

func TestNormalizeCreateRejectsManagedPaymentsForTestTransaction(t *testing.T) {
	in := createInputWithOptions(providers.PaymentOptions{Stripe: &providers.StripeOptions{
		MerchantOfRecord: "managed",
		ProductTaxCode:   "txcd_10000000",
	}})
	in.Test = true
	if _, _, _, _, err := normalizeCreate(in); !apperrors.Is(err, "provider_options_not_available_for_test") {
		t.Fatalf("normalizeCreate() = %v, want the test-mode policy rejection", err)
	}
}

func TestNormalizeCreateAcceptsProviderNeutralRequest(t *testing.T) {
	in := createInputWithOptions(providers.PaymentOptions{})
	in.Request.Adapter = constants.ProviderAuto.String()
	if _, _, _, _, err := normalizeCreate(in); err != nil {
		t.Fatalf("normalizeCreate() = %v", err)
	}
}

func TestHashRequestCoversProviderOptions(t *testing.T) {
	standard := CreateRequest{
		AmountMinor: 2500,
		Currency:    "USD",
		Adapter:     "stripe",
		Customer:    CustomerInput{Email: "payer@example.com"},
		Options: providers.PaymentOptions{Stripe: &providers.StripeOptions{
			ProductName: "Pro plan",
		}},
	}
	managed := standard
	managed.Options = providers.PaymentOptions{Stripe: &providers.StripeOptions{
		MerchantOfRecord: "managed",
		ProductTaxCode:   "txcd_10000000",
		ProductName:      "Pro plan",
	}}

	standardHash := hashRequest(2500, "USD", "ref_1", standard)
	if standardHash != hashRequest(2500, "USD", "ref_1", standard) {
		t.Fatal("hash is not deterministic")
	}
	if standardHash == hashRequest(2500, "USD", "ref_1", managed) {
		t.Fatal("a different provider option must change the request hash")
	}
}
