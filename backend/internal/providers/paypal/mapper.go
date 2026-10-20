package paypal

import (
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

// supportedCurrencies are the transaction currencies PayPal accepts. A currency
// outside this set cannot be charged, so it is rejected before the API call
// rather than surfacing as an opaque provider error.
var supportedCurrencies = map[string]bool{
	"AUD": true, "BRL": true, "CAD": true, "CNY": true, "CZK": true,
	"DKK": true, "EUR": true, "HKD": true, "HUF": true, "ILS": true,
	"JPY": true, "MXN": true, "MYR": true, "NOK": true, "NZD": true,
	"PHP": true, "PLN": true, "GBP": true, "SGD": true, "SEK": true,
	"CHF": true, "TWD": true, "THB": true, "USD": true, "ZAR": true,
}

func supportsCurrency(currency string) bool {
	return supportedCurrencies[strings.ToUpper(currency)]
}

// mapOrderStatus normalizes an order status onto the PayConnect vocabulary.
func mapOrderStatus(status string) string {
	switch strings.ToUpper(status) {
	case "COMPLETED":
		return string(constants.PaymentSuccessful)
	case "APPROVED":
		return string(constants.PaymentProcessing)
	case "VOIDED":
		return string(constants.PaymentCancelled)
	case "CREATED", "SAVED", "PAYER_ACTION_REQUIRED":
		return string(constants.PaymentPending)
	default:
		return string(constants.PaymentPending)
	}
}

// mapRefundStatus normalizes a PayPal refund status.
func mapRefundStatus(status string) string {
	switch strings.ToUpper(status) {
	case "COMPLETED":
		return string(constants.RefundSuccessful)
	case "PENDING":
		return string(constants.RefundProcessing)
	case "FAILED", "CANCELLED", "CANCELED":
		return string(constants.RefundFailed)
	default:
		return string(constants.RefundPending)
	}
}

// amountMinor converts a PayPal decimal amount into integer minor units.
func amountMinor(amount money) int64 {
	minor, err := providers.ParseMajor(amount.Value, amount.CurrencyCode)
	if err != nil {
		return 0
	}
	return minor
}

// resultFromOrder converts an order into a domain result. A completed order
// reports its capture id in the provider response for later reconciliation.
func resultFromOrder(o order) *providers.PaymentResult {
	result := &providers.PaymentResult{
		ProviderReference: o.ID,
		Status:            mapOrderStatus(o.Status),
		Currency:          "",
		Channel:           "paypal",
		RawStatus:         o.Status,
		ProviderResponse:  map[string]any{"id": o.ID, "status": o.Status},
	}
	if len(o.PurchaseUnits) > 0 {
		unit := o.PurchaseUnits[0]
		result.AmountMinor = amountMinor(unit.Amount)
		result.Currency = strings.ToUpper(unit.Amount.CurrencyCode)
		if len(unit.Payments.Captures) > 0 {
			captured := unit.Payments.Captures[0]
			result.ProviderResponse["capture_id"] = captured.ID
			result.ProviderResponse["capture_status"] = captured.Status
		}
	}
	return result
}

// approvalURL returns the customer-facing approval link, if PayPal issued one.
func approvalURL(o order) string {
	for _, candidate := range o.Links {
		switch strings.ToLower(candidate.Rel) {
		case "approve", "payer-action":
			return candidate.Href
		}
	}
	return ""
}

// captureID returns the first capture id on an order, which is what a refund
// targets.
func captureID(o order) string {
	if len(o.PurchaseUnits) == 0 || len(o.PurchaseUnits[0].Payments.Captures) == 0 {
		return ""
	}
	return o.PurchaseUnits[0].Payments.Captures[0].ID
}

// mapWebhookEvent normalizes a PayPal event, returning the PayConnect event type
// and the payment status it implies (empty when it implies none).
func mapWebhookEvent(eventType string) (string, string) {
	switch strings.ToUpper(eventType) {
	case "PAYMENT.CAPTURE.COMPLETED":
		return constants.EventPaymentSucceeded, string(constants.PaymentSuccessful)
	case "PAYMENT.CAPTURE.PENDING":
		return constants.EventPaymentProcessing, string(constants.PaymentProcessing)
	case "PAYMENT.CAPTURE.DENIED", "PAYMENT.CAPTURE.DECLINED", "PAYMENT.CAPTURE.FAILED":
		return constants.EventPaymentFailed, string(constants.PaymentFailed)
	case "PAYMENT.CAPTURE.REFUNDED":
		return constants.EventRefundSucceeded, ""
	case "PAYMENT.CAPTURE.REVERSED":
		return constants.EventChargebackCreated, ""
	case "CHECKOUT.ORDER.APPROVED":
		return constants.EventPaymentProcessing, string(constants.PaymentProcessing)
	case "CHECKOUT.ORDER.COMPLETED":
		return constants.EventPaymentSucceeded, string(constants.PaymentSuccessful)
	case "CHECKOUT.ORDER.VOIDED":
		return constants.EventPaymentCancelled, string(constants.PaymentCancelled)
	default:
		return "provider.event", ""
	}
}
