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
	default:
		return apperrors.ProviderNotSupported(provider.String())
	}
	return nil
}

func validateProviderCredentialFields(provider constants.Provider, apiKey, contractCode string) error {
	if provider != constants.ProviderMonnify {
		return nil
	}
	if strings.TrimSpace(apiKey) == "" {
		return apperrors.Validation("provider_credentials_invalid", "credentials.api_key is required for Monnify.")
	}
	if strings.TrimSpace(contractCode) == "" {
		return apperrors.Validation("provider_credentials_invalid", "credentials.contract_code is required for Monnify.")
	}
	return nil
}

func defaultCurrency(provider constants.Provider) string {
	switch provider {
	case constants.ProviderFlutterwave:
		return "NGN"
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
