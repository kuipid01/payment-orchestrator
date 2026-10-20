package providers

import (
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
)

func TestValidateMonnifyCredentialsRequiresAPIKeyAndContractCode(t *testing.T) {
	if err := validateProviderCredentialFields(constants.ProviderMonnify, "", "", "contract"); err == nil {
		t.Fatal("expected missing API key to be rejected")
	}
	if err := validateProviderCredentialFields(constants.ProviderMonnify, "", "api-key", ""); err == nil {
		t.Fatal("expected missing contract code to be rejected")
	}
	if err := validateProviderCredentialFields(constants.ProviderMonnify, "", "api-key", "contract"); err != nil {
		t.Fatalf("valid Monnify credentials rejected: %v", err)
	}
}

func TestValidatePayPalCredentialsRequireClientID(t *testing.T) {
	if err := validateProviderCredentialFields(constants.ProviderPayPal, "", "", ""); err == nil {
		t.Fatal("expected missing PayPal client ID to be rejected")
	}
	if err := validateProviderCredentialFields(constants.ProviderPayPal, "client-id", "", ""); err != nil {
		t.Fatalf("valid PayPal credentials rejected: %v", err)
	}
}

func TestValidateStripeCredentialsMatchEnvironment(t *testing.T) {
	cases := []struct {
		name        string
		environment string
		secret      string
		wantErr     bool
	}{
		{"test key in test", constants.EnvironmentTest, "sk_test_1234567890abcdef", false},
		{"live key in live", constants.EnvironmentLive, "sk_live_1234567890abcdef", false},
		{"restricted test key", constants.EnvironmentTest, "rk_test_1234567890abcdef", false},
		{"live key in test", constants.EnvironmentTest, "sk_live_1234567890abcdef", true},
		{"test key in live", constants.EnvironmentLive, "sk_test_1234567890abcdef", true},
		{"publishable key", constants.EnvironmentTest, "pk_test_1234567890abcdef", true},
		{"webhook secret", constants.EnvironmentTest, "whsec_1234567890abcdef", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCredentialShape(constants.ProviderStripe, tc.environment, tc.secret)
			if tc.wantErr && err == nil {
				t.Fatal("expected the Stripe key to be rejected")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("valid Stripe key rejected: %v", err)
			}
		})
	}
}

func TestValidateWebhookSecretMatchesProviderShape(t *testing.T) {
	if err := validateWebhookSecret(constants.ProviderStripe, "sk_test_123"); err == nil {
		t.Fatal("expected a non-whsec Stripe value to be rejected")
	}
	if err := validateWebhookSecret(constants.ProviderStripe, "whsec_1234567890abcdef"); err != nil {
		t.Fatalf("valid Stripe signing secret rejected: %v", err)
	}
	if err := validateWebhookSecret(constants.ProviderPayPal, "WH-REG-1"); err != nil {
		t.Fatalf("valid PayPal webhook id rejected: %v", err)
	}
	if err := validateWebhookSecret(constants.ProviderPayPal, "not an id"); err == nil {
		t.Fatal("expected a whitespace-containing PayPal webhook id to be rejected")
	}
}

func TestWebhookSupportDistinguishesProviderSigningSecrets(t *testing.T) {
	stripe := constants.WebhookSupportFor(constants.ProviderStripe)
	if !stripe.RequiresWebhookSecret || stripe.SignatureHeader != "Stripe-Signature" {
		t.Fatalf("Stripe webhook support = %#v", stripe)
	}
	paypal := constants.WebhookSupportFor(constants.ProviderPayPal)
	if !paypal.RequiresWebhookSecret || paypal.SignatureHeader != "paypal-transmission-sig" {
		t.Fatalf("PayPal webhook support = %#v", paypal)
	}
}
