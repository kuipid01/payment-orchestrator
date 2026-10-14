package constants

// Error types used in the public error envelope. These describe the *class* of
// failure; the specific machine code lives in internal/errors/codes.go.
const (
	ErrorTypeAuthentication = "authentication_error"
	ErrorTypeInvalidRequest = "invalid_request_error"
	ErrorTypePermission     = "permission_error"
	ErrorTypeNotFound       = "not_found_error"
	ErrorTypeProvider       = "provider_error"
	ErrorTypeConflict       = "conflict_error"
	ErrorTypeRateLimit      = "rate_limit_error"
	ErrorTypeInternal       = "internal_error"
)

// Merchant lifecycle statuses.
const (
	MerchantActive    = "active"
	MerchantSuspended = "suspended"
	MerchantPending   = "pending"
	MerchantClosed    = "closed"
)

// Provider connection statuses.
const (
	ConnectionConnected    = "connected"
	ConnectionNotConnected = "not_connected"
	ConnectionError        = "error"
)
