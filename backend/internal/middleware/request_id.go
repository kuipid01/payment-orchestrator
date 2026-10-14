// Package middleware holds the transport-level concerns shared by every route:
// request identity, authentication, idempotency, rate limiting, recovery and
// request logging. None of them contain business logic.
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// RequestIDLocalsKey is where the request ID is stored on the Fiber context.
const RequestIDLocalsKey = "requestID"

// RequestID honours a well-formed client-supplied X-Request-ID and otherwise
// generates one. The value is echoed back on the response and used in logs,
// provider calls and webhook processing.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := strings.TrimSpace(c.Get(fiber.HeaderXRequestID))
		if !validRequestID(requestID) {
			requestID = gormx.NewID("req")
		}
		c.Locals(RequestIDLocalsKey, requestID)
		c.Set(fiber.HeaderXRequestID, requestID)
		return c.Next()
	}
}

// RequestIDValue returns the current request ID.
func RequestIDValue(c *fiber.Ctx) string {
	if id, ok := c.Locals(RequestIDLocalsKey).(string); ok {
		return id
	}
	return ""
}

func validRequestID(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') && char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}
