package moniepoint

import (
	"strconv"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

// mapStatus normalizes a Moniepoint status onto PayConnect's vocabulary.
func mapStatus(status string) string {
	switch strings.ToLower(status) {
	case "successful", "success", "completed", "settled", "approved":
		return string(constants.PaymentSuccessful)
	case "failed", "declined", "rejected":
		return string(constants.PaymentFailed)
	case "cancelled", "canceled", "reversed":
		return string(constants.PaymentCancelled)
	case "expired":
		return string(constants.PaymentExpired)
	case "pending", "processing", "initiated", "queued":
		return string(constants.PaymentProcessing)
	default:
		return string(constants.PaymentProcessing)
	}
}

// mapRefundStatus normalizes a refund status.
func mapRefundStatus(status string) string {
	switch strings.ToLower(status) {
	case "successful", "success", "completed", "processed":
		return string(constants.RefundSuccessful)
	case "failed", "declined":
		return string(constants.RefundFailed)
	case "pending", "processing", "queued":
		return string(constants.RefundProcessing)
	default:
		return string(constants.RefundPending)
	}
}

// resultFromData converts a Moniepoint payment payload into a domain result.
func resultFromData(data map[string]any, fallbackReference string, fallbackAmount int64, currency string) *providers.PaymentResult {
	reference := stringFrom(data, "reference")
	if reference == "" {
		reference = fallbackReference
	}
	status := stringFrom(data, "status")
	amount := amountFrom(data["amount"], currency)
	if amount == 0 {
		amount = fallbackAmount
	}
	result := &providers.PaymentResult{
		ProviderReference: firstNonEmpty(stringFrom(data, "id"), reference),
		Status:            mapStatus(status),
		AmountMinor:       amount,
		Currency:          firstNonEmpty(stringFrom(data, "currency"), currency),
		AuthorizationURL:  firstNonEmpty(stringFrom(data, "authorization_url"), stringFrom(data, "checkout_url")),
		RawStatus:         status,
		ProviderResponse: map[string]any{
			"status": status,
			"id":     stringFrom(data, "id"),
		},
	}
	if result.Status == string(constants.PaymentFailed) {
		result.FailureCode = "provider_declined"
		result.FailureMessage = firstNonEmpty(stringFrom(data, "message"), stringFrom(data, "reason"))
	}
	return result
}

// parseLoose parses a JSON number via its textual representation.
func parseLoose(value float64, currency string) int64 {
	minor, err := providers.ParseMajor(strconv.FormatFloat(value, 'f', 2, 64), currency)
	if err != nil {
		return 0
	}
	return minor
}

func amountFrom(value any, currency string) int64 {
	switch typed := value.(type) {
	case float64:
		// JSON numbers decode as float64; only the textual form is parsed so no
		// floating point value ever becomes money.
		return parseLoose(typed, currency)
	case string:
		if minor, err := providers.ParseMajor(typed, currency); err == nil {
			return minor
		}
		return 0
	case int64:
		return typed
	default:
		return 0
	}
}

func stringFrom(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
