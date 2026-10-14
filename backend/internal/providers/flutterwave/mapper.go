package flutterwave

import (
	"log/slog"
	"strconv"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

// mapStatus normalizes a Flutterwave status onto PayConnect's vocabulary.
func mapStatus(status string) string {
	switch status {
	case "successful", "success", "completed":
		return string(constants.PaymentSuccessful)
	case "failed":
		return string(constants.PaymentFailed)
	case "cancelled", "canceled":
		return string(constants.PaymentCancelled)
	case "pending", "processing":
		return string(constants.PaymentProcessing)
	case "reversed", "refunded":
		return string(constants.PaymentRefunded)
	case "expired":
		return string(constants.PaymentExpired)
	default:
		return string(constants.PaymentProcessing)
	}
}

// mapRefundStatus normalizes a refund status.
func mapRefundStatus(status string) string {
	switch status {
	case "completed", "successful", "success":
		return string(constants.RefundSuccessful)
	case "failed":
		return string(constants.RefundFailed)
	case "pending", "processing", "queued":
		return string(constants.RefundProcessing)
	default:
		return string(constants.RefundPending)
	}
}

// amountFrom converts Flutterwave's loosely typed amount (number or string)
// into integer minor units.
func amountFrom(value any, currency string) int64 {
	var raw string
	switch typed := value.(type) {
	case string:
		raw = typed
	case float64:
		// JSON numbers arrive as float64; format without introducing precision
		// loss for typical amounts, then parse as a decimal string.
		raw = strconv.FormatFloat(typed, 'f', 2, 64)
	case int64:
		raw = strconv.FormatInt(typed, 10)
	default:
		return 0
	}
	minor, err := providers.ParseMajor(raw, currency)
	if err != nil {
		slog.Warn("flutterwave: could not parse amount", "value", raw, "currency", currency)
		return 0
	}
	return minor
}

// resultFromTransaction converts a Flutterwave transaction into a domain result.
func resultFromTransaction(tx transaction) *providers.PaymentResult {
	currency := tx.Currency
	if currency == "" {
		currency = constants.DefaultCurrency
	}
	result := &providers.PaymentResult{
		ProviderReference: firstNonEmpty(tx.FlwRef, tx.TxRef),
		Status:            mapStatus(tx.Status),
		AmountMinor:       amountFrom(tx.ChargedAmount, currency),
		Currency:          currency,
		RawStatus:         tx.Status,
		ProviderResponse: map[string]any{
			"id":      tx.ID,
			"status":  tx.Status,
			"flw_ref": tx.FlwRef,
		},
	}
	if result.AmountMinor == 0 {
		result.AmountMinor = amountFrom(tx.Amount, currency)
	}
	if result.Status == string(constants.PaymentFailed) {
		result.FailureCode = "provider_declined"
		result.FailureMessage = tx.ProcessorResponse
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
