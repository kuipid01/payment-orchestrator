package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// IdempotencyKeyLocalsKey is where the validated Idempotency-Key is stored.
const IdempotencyKeyLocalsKey = "idempotencyKey"

// MaxIdempotencyKeyLength bounds the header so it cannot be used as a payload.
const MaxIdempotencyKeyLength = 255

// Idempotency validates and captures the Idempotency-Key header.
//
// The dedupe itself belongs to the service (it needs the request hash and the
// stored response), so this middleware only guarantees that unsafe operations
// carry a well-formed key before they reach business logic.
func Idempotency(required bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := strings.TrimSpace(c.Get("Idempotency-Key"))
		if key == "" {
			if required {
				return apperrors.IdempotencyKeyRequired()
			}
			return c.Next()
		}
		if len(key) > MaxIdempotencyKeyLength || !validIdempotencyKey(key) {
			return apperrors.Validation(apperrors.CodeValidationError,
				"Idempotency-Key must be 1-255 characters of letters, digits, '-', '_' or '.'.")
		}
		c.Locals(IdempotencyKeyLocalsKey, key)
		return c.Next()
	}
}

// IdempotencyKey returns the validated idempotency key ("" when absent).
func IdempotencyKey(c *fiber.Ctx) string {
	if key, ok := c.Locals(IdempotencyKeyLocalsKey).(string); ok {
		return key
	}
	return strings.TrimSpace(c.Get("Idempotency-Key"))
}

func validIdempotencyKey(value string) bool {
	for _, ch := range value {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') &&
			(ch < '0' || ch > '9') && ch != '-' && ch != '_' && ch != '.' {
			return false
		}
	}
	return true
}
