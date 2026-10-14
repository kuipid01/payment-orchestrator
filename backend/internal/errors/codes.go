package apperrors

// Machine-readable error codes. Every code here maps to exactly one HTTP status
// via the Default* constructors below, so clients can switch on the code.
const (
	// Authentication & merchant state.
	CodeAuthenticationRequired = "authentication_required"
	CodeInvalidAPIKey          = "invalid_api_key"
	CodeAPIKeyRevoked          = "api_key_revoked"
	CodeAPIKeyExpired          = "api_key_expired"
	CodeMerchantSuspended      = "merchant_suspended"
	CodePermissionDenied       = "permission_denied"
	CodeInvalidCredentials     = "invalid_credentials"
	CodeEmailNotVerified       = "email_not_verified"
	CodeTokenInvalid           = "token_invalid"

	// Validation.
	CodeValidationError      = "validation_error"
	CodeInvalidAmount        = "invalid_amount"
	CodeInvalidCurrency      = "invalid_currency"
	CodeInvalidReference     = "invalid_reference"
	CodeInvalidEnvironment   = "invalid_environment"
	CodeInvalidProvider      = "invalid_provider"
	CodeInvalidRequest       = "invalid_request"
	CodeInvalidStatus        = "invalid_status"
	CodeInvalidWebhookTarget = "invalid_webhook_target"

	// Providers.
	CodeProviderNotConnected        = "provider_not_connected"
	CodeProviderNotSupported        = "provider_not_supported"
	CodeProviderUnavailable         = "provider_unavailable"
	CodeProviderAuthenticationFails = "provider_authentication_failed"
	CodeProviderRequestFailed       = "provider_request_failed"
	CodeProviderTimeout             = "provider_timeout"
	CodeProviderCredentialsInvalid  = "provider_credentials_invalid"

	// Payments.
	CodePaymentNotFound         = "payment_not_found"
	CodePaymentAlreadyProcessed = "payment_already_processed"
	CodePaymentNotRefundable    = "payment_not_refundable"
	CodePaymentAlreadyRefunded  = "payment_already_refunded"
	CodePaymentNotVerifiable    = "payment_not_verifiable"
	CodeReferenceAlreadyUsed    = "reference_already_used"

	// Refunds.
	CodeRefundNotFound     = "refund_not_found"
	CodeRefundFailed       = "refund_failed"
	CodeRefundAmountTooBig = "refund_amount_exceeds_payment"

	// Idempotency.
	CodeIdempotencyKeyRequired = "idempotency_key_required"
	CodeIdempotencyKeyReused   = "idempotency_key_reused"
	CodeIdempotencyConflict    = "idempotency_conflict"

	// Webhooks.
	CodeWebhookSignatureInvalid = "webhook_signature_invalid"
	CodeWebhookEventInvalid     = "webhook_event_invalid"
	CodeWebhookAlreadyProcessed = "webhook_already_processed"
	CodeWebhookEndpointNotFound = "webhook_endpoint_not_found"
	CodeWebhookDeliveryNotFound = "webhook_delivery_not_found"

	// Customers & routing.
	CodeCustomerNotFound = "customer_not_found"
	CodeRouteNotFound    = "route_not_found"

	// Billing & subscriptions.
	CodePlanNotFound         = "plan_not_found"
	CodeSubscriptionNotFound = "subscription_not_found"
	CodeTrialAlreadyUsed     = "trial_already_used"
	CodeBillingNotConfigured = "billing_not_configured"
	// CodeUpgradeRequired is the wall: the caller's plan does not include what
	// they just asked for, and the details name the plan, the metric and the
	// limit so the UI can prompt in place instead of guessing.
	CodeUpgradeRequired = "upgrade_required"

	// Infrastructure.
	CodeRateLimitExceeded  = "rate_limit_exceeded"
	CodeNotFound           = "not_found"
	CodeInternalError      = "internal_error"
	CodeServiceUnavailable = "service_unavailable"
)
