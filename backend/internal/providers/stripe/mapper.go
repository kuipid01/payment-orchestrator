package stripe

import (
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

// mapSessionStatus normalizes a Checkout Session onto the PayConnect status
// vocabulary. Provider statuses never reach the public API.
func mapSessionStatus(s session) string {
	switch strings.ToLower(s.PaymentStatus) {
	case "paid", "no_payment_required":
		return string(constants.PaymentSuccessful)
	case "unpaid":
		if strings.EqualFold(s.Status, "expired") {
			return string(constants.PaymentExpired)
		}
		if strings.EqualFold(s.Status, "complete") {
			// Complete but unpaid: an asynchronous method is still settling.
			return string(constants.PaymentProcessing)
		}
		return string(constants.PaymentPending)
	}
	if strings.EqualFold(s.Status, "expired") {
		return string(constants.PaymentExpired)
	}
	if strings.EqualFold(s.Status, "complete") {
		return string(constants.PaymentProcessing)
	}
	return string(constants.PaymentPending)
}

// mapRefundStatus normalizes a Stripe refund status.
func mapRefundStatus(status string) string {
	switch strings.ToLower(status) {
	case "succeeded":
		return string(constants.RefundSuccessful)
	case "pending", "requires_action":
		return string(constants.RefundProcessing)
	case "failed", "canceled", "cancelled":
		return string(constants.RefundFailed)
	default:
		return string(constants.RefundPending)
	}
}

// resultFromSession converts a Checkout Session into a domain result.
func resultFromSession(s session) *providers.PaymentResult {
	result := &providers.PaymentResult{
		ProviderReference: s.ID,
		Status:            mapSessionStatus(s),
		AmountMinor:       s.AmountTotal,
		Currency:          strings.ToUpper(s.Currency),
		AuthorizationURL:  s.URL,
		Channel:           "card",
		RawStatus:         firstNonEmpty(s.PaymentStatus, s.Status),
		ProviderResponse: map[string]any{
			"id":             s.ID,
			"status":         s.Status,
			"payment_status": s.PaymentStatus,
			"payment_intent": s.PaymentIntent,
			"amount_total":   s.AmountTotal,
			"currency":       s.Currency,
		},
	}
	if result.Status == string(constants.PaymentExpired) {
		result.FailureCode = "checkout_expired"
		result.FailureMessage = "The Stripe checkout session expired before it was paid."
	}
	return result
}

// mapEventType normalizes a Stripe event onto a PayConnect event type. It needs
// the session because checkout.session.completed carries a payment status.
func mapEventType(eventType string, s session) string {
	switch strings.ToLower(eventType) {
	case "checkout.session.completed":
		if mapSessionStatus(s) == string(constants.PaymentSuccessful) {
			return constants.EventPaymentSucceeded
		}
		return constants.EventPaymentProcessing
	case "checkout.session.async_payment_succeeded":
		return constants.EventPaymentSucceeded
	case "checkout.session.async_payment_failed":
		return constants.EventPaymentFailed
	case "checkout.session.expired":
		return constants.EventPaymentFailed
	case "charge.refunded":
		return constants.EventRefundSucceeded
	case "charge.dispute.created", "charge.dispute.updated":
		return constants.EventChargebackCreated
	default:
		return "provider.event"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
