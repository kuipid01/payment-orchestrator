package providers

import "testing"

// Descriptors is what GET /v1/providers serves, so a provider missing here is
// invisible to the dashboard even though its adapter works.
func TestDescriptorsAdvertiseStripeAndPayPalWebhookContracts(t *testing.T) {
	byName := make(map[string]ProviderDescriptor)
	for _, descriptor := range Descriptors() {
		byName[descriptor.Provider] = descriptor
	}

	// Only a merchant-chosen signing value may be entered before the PayConnect
	// endpoint exists. Stripe's whsec_ and PayPal's webhook id are issued for a
	// registered endpoint, so the connect form must not ask for them upfront.
	if byName["flutterwave"].WebhookSecretSelfAssigned != true {
		t.Fatalf("flutterwave descriptor should be self-assigned: %#v", byName["flutterwave"])
	}
	if byName["stripe"].WebhookSecretSelfAssigned || byName["paypal"].WebhookSecretSelfAssigned {
		t.Fatal("stripe and paypal webhook values are issued by the provider")
	}

	stripe, ok := byName["stripe"]
	if !ok {
		t.Fatal("stripe is not advertised")
	}
	if !stripe.WebhookSecretRequired || stripe.WebhookSignatureHeader != "Stripe-Signature" {
		t.Fatalf("stripe descriptor = %#v", stripe)
	}

	paypal, ok := byName["paypal"]
	if !ok {
		t.Fatal("paypal is not advertised")
	}
	if !paypal.WebhookSecretRequired || paypal.WebhookSecretLabel != "Webhook ID" {
		t.Fatalf("paypal descriptor = %#v", paypal)
	}
}
