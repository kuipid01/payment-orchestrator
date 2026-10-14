package apihttp

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// RequestID returns the request ID assigned by the request-id middleware.
func RequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals("requestID").(string); ok {
		return id
	}
	return c.Get(fiber.HeaderXRequestID)
}

// IdempotencyKey returns the Idempotency-Key header ("" when absent).
func IdempotencyKey(c *fiber.Ctx) string {
	return c.Get("Idempotency-Key")
}

// Body decodes the JSON request body, returning a typed validation error on
// malformed input. Unknown fields are ignored on purpose (forward compatibility).
func Body(c *fiber.Ctx, into any) error {
	if len(c.Body()) == 0 {
		return apperrors.Validation(apperrors.CodeInvalidRequest, "A JSON request body is required.")
	}
	if err := json.Unmarshal(c.Body(), into); err != nil {
		return apperrors.Validation(apperrors.CodeInvalidRequest, "The request body is not valid JSON.")
	}
	return nil
}

// Query returns a query parameter or a fallback.
func Query(c *fiber.Ctx, key, fallback string) string {
	if v := c.Query(key); v != "" {
		return v
	}
	return fallback
}
