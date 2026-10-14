package apperrors

import "net/http"

/* ------------------------- typed constructors -------------------------------- */
// Each helper pins the envelope class, the machine code and the HTTP status so
// call sites cannot invent inconsistent combinations.

func AuthenticationRequired() *AppError {
	return New("authentication_error", CodeAuthenticationRequired, "Authentication is required.", http.StatusUnauthorized)
}

func InvalidAPIKey() *AppError {
	return New("authentication_error", CodeInvalidAPIKey, "The API key provided is invalid.", http.StatusUnauthorized)
}

func APIKeyRevoked() *AppError {
	return New("authentication_error", CodeAPIKeyRevoked, "This API key has been revoked.", http.StatusUnauthorized)
}

func APIKeyExpired() *AppError {
	return New("authentication_error", CodeAPIKeyExpired, "This API key has expired.", http.StatusUnauthorized)
}

func MerchantSuspended() *AppError {
	return New("permission_error", CodeMerchantSuspended, "This merchant account is not active.", http.StatusForbidden)
}

func PermissionDenied(message string) *AppError {
	if message == "" {
		message = "You do not have permission to perform this action."
	}
	return New("permission_error", CodePermissionDenied, message, http.StatusForbidden)
}

func InvalidCredentials() *AppError {
	return New("authentication_error", CodeInvalidCredentials, "Email or password is incorrect.", http.StatusUnauthorized)
}

func EmailNotVerified() *AppError {
	return New("authentication_error", CodeEmailNotVerified, "Verify your email before signing in.", http.StatusForbidden)
}

func TokenInvalid(message string) *AppError {
	if message == "" {
		message = "The token is invalid or has expired."
	}
	return New("authentication_error", CodeTokenInvalid, message, http.StatusBadRequest)
}

func Validation(code, message string) *AppError {
	if code == "" {
		code = CodeValidationError
	}
	return New("invalid_request_error", code, message, http.StatusBadRequest)
}

func InvalidAmount(message string) *AppError {
	return New("invalid_request_error", CodeInvalidAmount, message, http.StatusBadRequest)
}

func InvalidCurrency(message string) *AppError {
	return New("invalid_request_error", CodeInvalidCurrency, message, http.StatusBadRequest)
}

func InvalidEnvironment() *AppError {
	return New("invalid_request_error", CodeInvalidEnvironment, "environment must be test or live.", http.StatusBadRequest)
}

func InvalidProvider(message string) *AppError {
	if message == "" {
		message = "The requested provider is not supported."
	}
	return New("invalid_request_error", CodeInvalidProvider, message, http.StatusBadRequest)
}

func NotFound(code, message string) *AppError {
	if code == "" {
		code = CodeNotFound
	}
	return New("not_found_error", code, message, http.StatusNotFound)
}

func Conflict(code, message string) *AppError {
	return New("conflict_error", code, message, http.StatusConflict)
}

func ProviderNotConnected(provider string) *AppError {
	return New("provider_error", CodeProviderNotConnected,
		"No connected "+provider+" credentials for this environment.", http.StatusBadRequest)
}

func ProviderNotSupported(provider string) *AppError {
	return New("provider_error", CodeProviderNotSupported, "Provider "+provider+" is not supported.", http.StatusBadRequest)
}

func ProviderUnavailable(provider string, cause error) *AppError {
	return New("provider_error", CodeProviderUnavailable,
		"The selected payment provider is temporarily unavailable.", http.StatusBadGateway).WithCause(cause)
}

func ProviderTimeout(provider string, cause error) *AppError {
	return New("provider_error", CodeProviderTimeout,
		provider+" did not respond in time. The payment was not confirmed.", http.StatusGatewayTimeout).WithCause(cause)
}

func ProviderRequestFailed(provider string, cause error) *AppError {
	return New("provider_error", CodeProviderRequestFailed,
		"The payment provider rejected the request.", http.StatusBadGateway).WithCause(cause)
}

func ProviderAuthenticationFailed(provider string, cause error) *AppError {
	return New("provider_error", CodeProviderAuthenticationFails,
		"The stored "+provider+" credentials were rejected.", http.StatusBadGateway).WithCause(cause)
}

func PaymentNotFound() *AppError {
	return New("not_found_error", CodePaymentNotFound, "Payment not found.", http.StatusNotFound)
}

func PaymentNotRefundable(message string) *AppError {
	if message == "" {
		message = "This payment cannot be refunded."
	}
	return New("conflict_error", CodePaymentNotRefundable, message, http.StatusConflict)
}

func PaymentAlreadyProcessed() *AppError {
	return New("conflict_error", CodePaymentAlreadyProcessed, "This payment has already been processed.", http.StatusConflict)
}

func ReferenceAlreadyUsed(reference string) *AppError {
	return New("conflict_error", CodeReferenceAlreadyUsed,
		"The reference "+reference+" has already been used by another payment.", http.StatusConflict)
}

func RefundNotFound() *AppError {
	return New("not_found_error", CodeRefundNotFound, "Refund not found.", http.StatusNotFound)
}

func RefundFailed(message string, cause error) *AppError {
	if message == "" {
		message = "The refund could not be completed."
	}
	return New("provider_error", CodeRefundFailed, message, http.StatusBadGateway).WithCause(cause)
}

func IdempotencyKeyRequired() *AppError {
	return New("invalid_request_error", CodeIdempotencyKeyRequired,
		"An Idempotency-Key header is required when creating a payment.", http.StatusBadRequest)
}

func IdempotencyKeyReused() *AppError {
	return New("conflict_error", CodeIdempotencyKeyReused,
		"This Idempotency-Key was already used with a different request body.", http.StatusConflict)
}

func IdempotencyConflict() *AppError {
	return New("conflict_error", CodeIdempotencyConflict,
		"A request with this Idempotency-Key is still being processed.", http.StatusConflict)
}

func WebhookSignatureInvalid() *AppError {
	return New("authentication_error", CodeWebhookSignatureInvalid, "Webhook signature verification failed.", http.StatusUnauthorized)
}

func WebhookEventInvalid(message string) *AppError {
	if message == "" {
		message = "The webhook payload could not be parsed."
	}
	return New("invalid_request_error", CodeWebhookEventInvalid, message, http.StatusBadRequest)
}

func CustomerNotFound() *AppError {
	return New("not_found_error", CodeCustomerNotFound, "Customer not found.", http.StatusNotFound)
}

func RouteNotFound() *AppError {
	return New("not_found_error", CodeRouteNotFound, "Routing rule not found.", http.StatusNotFound)
}

func RateLimitExceeded() *AppError {
	return New("rate_limit_error", CodeRateLimitExceeded, "Too many requests.", http.StatusTooManyRequests)
}

func ServiceUnavailable(message string, cause error) *AppError {
	if message == "" {
		message = "The service is temporarily unavailable."
	}
	return New("internal_error", CodeServiceUnavailable, message, http.StatusServiceUnavailable).WithCause(cause)
}
