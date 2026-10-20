package providers

import (
	"errors"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// validateCredentialShape is a fast local pre-flight. It rejects obviously wrong
// credentials before a network round-trip; the provider remains the authority.
func validateCredentialShape(provider constants.Provider, environment, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return apperrors.Validation("provider_credentials_invalid", "credentials.secret_key is required.")
	}
	if len(secret) < 20 {
		return apperrors.Validation("provider_credentials_invalid", "credentials.secret_key looks too short.")
	}
	switch provider {
	case constants.ProviderPaystack:
		if !strings.HasPrefix(secret, "sk_"+environment+"_") {
			return apperrors.Validation("provider_credentials_invalid",
				"Paystack secret keys look like sk_"+environment+"_…")
		}
	case constants.ProviderFlutterwave:
		if !strings.HasPrefix(secret, "FLWSECK") {
			return apperrors.Validation("provider_credentials_invalid", "Flutterwave secret keys look like FLWSECK-…")
		}
	case constants.ProviderMonnify:
		if strings.ContainsAny(secret, " \t\n") {
			return apperrors.Validation("provider_credentials_invalid", "credentials.secret_key must not contain whitespace.")
		}
	case constants.ProviderStripe:
		switch {
		case environment == constants.EnvironmentLive && (strings.HasPrefix(secret, "sk_live_") || strings.HasPrefix(secret, "rk_live_")):
		case environment == constants.EnvironmentTest && (strings.HasPrefix(secret, "sk_test_") || strings.HasPrefix(secret, "rk_test_")):
		case strings.HasPrefix(secret, "pk_"):
			return apperrors.Validation("provider_credentials_invalid",
				"That is a Stripe publishable key (pk_…); PayConnect needs the secret key (sk_…).")
		case strings.HasPrefix(secret, "whsec_"):
			return apperrors.Validation("provider_credentials_invalid",
				"That is a Stripe webhook signing secret; PayConnect needs the account secret key (sk_…).")
		default:
			return apperrors.Validation("provider_credentials_invalid",
				"Stripe secret keys look like sk_"+environment+"_…; the key must match the connection environment.")
		}
	case constants.ProviderPayPal:
		if strings.ContainsAny(secret, " \t\n") {
			return apperrors.Validation("provider_credentials_invalid", "credentials.secret_key must not contain whitespace.")
		}
	default:
		return apperrors.ProviderNotSupported(provider.String())
	}
	return nil
}

func validateProviderCredentialFields(provider constants.Provider, publicKey, apiKey, contractCode string) error {
	switch provider {
	case constants.ProviderMonnify:
		if strings.TrimSpace(apiKey) == "" {
			return apperrors.Validation("provider_credentials_invalid", "credentials.api_key is required for Monnify.")
		}
		if strings.TrimSpace(contractCode) == "" {
			return apperrors.Validation("provider_credentials_invalid", "credentials.contract_code is required for Monnify.")
		}
	case constants.ProviderPayPal:
		// PayPal authenticates with a client id and a client secret; the id is
		// public, so it travels in public_key alongside the secret.
		if strings.TrimSpace(publicKey) == "" {
			return apperrors.Validation("provider_credentials_invalid", "credentials.public_key is required for PayPal (the REST app client ID).")
		}
	}
	return nil
}

// validateWebhookSecret is a local shape check for the inbound webhook value,
// so a mistyped dashboard paste is rejected at save time instead of at the
// first real delivery. The provider remains the authority on the secret.
func validateWebhookSecret(provider constants.Provider, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return apperrors.Validation(apperrors.CodeValidationError, "webhook_secret is required.")
	}
	switch provider {
	case constants.ProviderStripe:
		if !strings.HasPrefix(secret, "whsec_") {
			return apperrors.Validation("provider_webhook_secret_invalid",
				"Stripe webhook signing secrets look like whsec_… (Developers → Webhooks → your endpoint → Signing secret).")
		}
	case constants.ProviderPayPal:
		if strings.ContainsAny(secret, " \t\n") {
			return apperrors.Validation("provider_webhook_secret_invalid",
				"A PayPal webhook ID must not contain whitespace.")
		}
		if len(secret) < 8 {
			return apperrors.Validation("provider_webhook_secret_invalid",
				"That does not look like a PayPal webhook ID.")
		}
	}
	return nil
}

func defaultCurrency(provider constants.Provider) string {
	switch provider {
	case constants.ProviderStripe, constants.ProviderPayPal:
		return "USD"
	default:
		return "NGN"
	}
}

func last4(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 4 {
		return value
	}
	return value[len(value)-4:]
}

var errNotConnected = errors.New("providers: connection is not connected")
