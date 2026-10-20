package constants

// Provider identifies a payment provider integration.
type Provider string

const (
	ProviderPaystack    Provider = "paystack"
	ProviderFlutterwave Provider = "flutterwave"
	ProviderMonnify     Provider = "monnify"
	ProviderStripe      Provider = "stripe"
	ProviderPayPal      Provider = "paypal"

	// ProviderAuto asks the routing service to pick a provider.
	ProviderAuto Provider = "auto"
)

// AllProviders lists every provider With a registry entry.
func AllProviders() []Provider {
	return []Provider{ProviderPaystack, ProviderFlutterwave, ProviderMonnify, ProviderStripe, ProviderPayPal}
}

// IsValidProvider reports whether p is a concrete, supported provider.
func IsValidProvider(p Provider) bool {
	for _, known := range AllProviders() {
		if p == known {
			return true
		}
	}
	return false
}

// String keeps the value usable wherever a string is expected.
func (p Provider) String() string { return string(p) }

// WebhookSupport describes how a provider signs its inbound webhooks and what a
// merchant must configure for verification to succeed. It is the single source
// of truth the API and UI read, so provider differences never leak into
// handlers or components.
type WebhookSupport struct {
	// SignatureHeader is the request header carrying the signature.
	SignatureHeader string
	// RequiresWebhookSecret is true when the provider signs with a value the
	// merchant defines in the provider dashboard, distinct from the API secret
	// key. When false, the connection's own secret key is the signing key and
	// no extra configuration is needed.
	RequiresWebhookSecret bool
	// SecretLabel names that value in the provider's own vocabulary.
	SecretLabel string
	// SecretSelfAssigned is true when the merchant chooses the signing value
	// before any endpoint exists. When false, the provider issues the value for
	// a registered endpoint, so it can only be supplied after the PayConnect
	// webhook URL has been created.
	SecretSelfAssigned bool
	// DashboardPath is where the merchant configures the webhook.
	DashboardPath string
}

// WebhookSupportFor returns the inbound webhook contract for a provider.
func WebhookSupportFor(p Provider) WebhookSupport {
	switch p {
	case ProviderPaystack:
		// Paystack signs with the secret key (HMAC-SHA512): paste the URL only.
		return WebhookSupport{
			SignatureHeader: "X-Paystack-Signature",
			DashboardPath:   "Settings → API Keys & Webhooks",
		}
	case ProviderFlutterwave:
		// Flutterwave authenticates with a merchant-defined "verif-hash" set in
		// the dashboard, so a webhook secret distinct from the API key is needed.
		return WebhookSupport{
			SignatureHeader:       "verif-hash",
			RequiresWebhookSecret: true,
			SecretLabel:           "Secret hash",
			SecretSelfAssigned:    true,
			DashboardPath:         "Settings → Webhooks",
		}
	case ProviderMonnify:
		return WebhookSupport{
			SignatureHeader: "monnify-signature",
			DashboardPath:   "Settings → API Keys & Webhooks",
		}
	case ProviderStripe:
		// Stripe signs with a per-endpoint signing secret created in the
		// dashboard, never with the account's API key.
		return WebhookSupport{
			SignatureHeader:       "Stripe-Signature",
			RequiresWebhookSecret: true,
			SecretLabel:           "Webhook signing secret",
			DashboardPath:         "Developers → Webhooks",
		}
	case ProviderPayPal:
		// PayPal verifies deliveries server-to-server against a webhook
		// registration, identified by its webhook id.
		return WebhookSupport{
			SignatureHeader:       "paypal-transmission-sig",
			RequiresWebhookSecret: true,
			SecretLabel:           "Webhook ID",
			DashboardPath:         "Developer Dashboard → Webhooks",
		}
	default:
		return WebhookSupport{}
	}
}
