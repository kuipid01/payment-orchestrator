package constants

// Merchant webhook event types.
const (
	EventPaymentCreated    = "payment.created"
	EventPaymentProcessing = "payment.processing"
	EventPaymentSucceeded  = "payment.succeeded"
	EventPaymentFailed     = "payment.failed"
	EventPaymentCancelled  = "payment.cancelled"

	EventRefundCreated   = "refund.created"
	EventRefundSucceeded = "refund.succeeded"
	EventRefundFailed    = "refund.failed"

	EventChargebackCreated = "chargeback.created"
	EventChargebackUpdated = "chargeback.updated"
)

// Webhook delivery statuses.
const (
	DeliveryPending   = "pending"
	DeliveryRetrying  = "retrying"
	DeliveryDelivered = "delivered"
	DeliveryFailed    = "failed"
	DeliveryDead      = "dead"
)

// WebhookPayloadVersion identifies the merchant webhook body contract. Every
// payload carries it in its meta block, so a consumer can rely on one shape and
// one place for delivery facts no matter which provider processed the payment.
// Bump it only for a breaking change to the envelope.
const WebhookPayloadVersion = "v1"

// Webhook signature headers used for merchant deliveries.
const (
	HeaderSignature  = "X-PayConnect-Signature"
	HeaderEvent      = "X-PayConnect-Event"
	HeaderDeliveryID = "X-PayConnect-Delivery-ID"
)

// AllMerchantEvents lists the events a merchant endpoint may subscribe to.
func AllMerchantEvents() []string {
	return []string{
		EventPaymentCreated, EventPaymentProcessing, EventPaymentSucceeded,
		EventPaymentFailed, EventPaymentCancelled,
		EventRefundCreated, EventRefundSucceeded, EventRefundFailed,
		EventChargebackCreated, EventChargebackUpdated,
	}
}

// IsMerchantEvent reports whether name is a subscribable event.
func IsMerchantEvent(name string) bool {
	for _, e := range AllMerchantEvents() {
		if e == name {
			return true
		}
	}
	return false
}
