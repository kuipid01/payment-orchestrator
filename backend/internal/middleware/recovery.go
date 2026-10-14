package middleware

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// Recovery converts a panic into the standard error envelope instead of
// crashing the process or leaking a stack trace to the client.
func Recovery(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if log != nil {
					log.Error("panic recovered",
						"path", c.Path(), "method", c.Method(), "request_id", RequestIDValue(c), "panic", r)
				}
				err = apperrors.Internal(nil).WithMessage("An unexpected error occurred.")
			}
		}()
		return c.Next()
	}
}
