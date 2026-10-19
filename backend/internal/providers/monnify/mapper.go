package monnify

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

func mapPaymentStatus(status string) string {
	switch strings.ToUpper(status) {
	case "PAID", "SUCCESS", "SUCCESSFUL", "COMPLETED":
		return string(constants.PaymentSuccessful)
	case "FAILED", "REJECTED":
		return string(constants.PaymentFailed)
	case "CANCELLED", "CANCELED":
		return string(constants.PaymentCancelled)
	case "EXPIRED":
		return string(constants.PaymentExpired)
	case "PENDING", "UNPAID", "PROCESSING":
		return string(constants.PaymentProcessing)
	default:
		return string(constants.PaymentProcessing)
	}
}

func mapRefundStatus(status string) string {
	switch strings.ToUpper(status) {
	case "COMPLETED", "SUCCESSFUL", "SUCCESS":
		return string(constants.RefundSuccessful)
	case "FAILED", "REJECTED":
		return string(constants.RefundFailed)
	case "PROCESSING", "PENDING", "INITIATED":
		return string(constants.RefundProcessing)
	default:
		return string(constants.RefundPending)
	}
}

func amountFrom(value any, currency string, fallback int64) int64 {
	var raw string
	switch typed := value.(type) {
	case string:
		raw = typed
	case float64:
		raw = strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		raw = string(typed)
	case int64:
		raw = strconv.FormatInt(typed, 10)
	default:
		return fallback
	}
	amount, err := providers.ParseMajor(raw, currency)
	if err != nil {
		return fallback
	}
	return amount
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
