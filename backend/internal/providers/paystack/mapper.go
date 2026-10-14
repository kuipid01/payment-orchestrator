package paystack

import (
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

// mapStatus normalizes a Paystack status onto the PayConnect status vocabulary.
// Provider statuses never reach the public API.
func mapStatus(status string) string {
	switch strings.ToLower(status) {
	case "success", "successful", "completed", "processed":
		return string(constants.PaymentSuccessful)
	case "failed":
		return string(constants.PaymentFailed)
	case "abandoned", "cancelled", "canceled":
		return string(constants.PaymentCancelled)
	case "reversed":
		return string(constants.PaymentRefunded)
	case "pending", "ongoing", "queued":
		return string(constants.PaymentProcessing)
	case "expired":
		return string(constants.PaymentExpired)
	default:
		return string(constants.PaymentProcessing)
	}
}

// mapRefundStatus normalizes a refund status.
func mapRefundStatus(status string) string {
	switch strings.ToLower(status) {
	case "processed", "success", "successful":
		return string(constants.RefundSuccessful)
	case "failed":
		return string(constants.RefundFailed)
	case "pending", "processing", "queued":
		return string(constants.RefundProcessing)
	default:
		return string(constants.RefundPending)
	}
}

// resultFromData converts a Paystack transaction payload into a domain result.
func resultFromData(data transactionData) *providers.PaymentResult {
	result := &providers.PaymentResult{
		ProviderReference: data.Reference,
		Status:            mapStatus(data.Status),
		AmountMinor:       data.Amount,
		Currency:          data.Currency,
		AuthorizationURL:  data.AuthorizationURL,
		Channel:           data.Channel,
		FeeMinor:          data.Fees,
		RawStatus:         data.Status,
		ProviderResponse: map[string]any{
			"id":               data.ID,
			"status":           data.Status,
			"gateway_response": data.GatewayResponse,
			"channel":          data.Channel,
		},
	}
	if result.Status == string(constants.PaymentFailed) {
		result.FailureCode = "provider_declined"
		result.FailureMessage = data.GatewayResponse
	}
	return result
}
