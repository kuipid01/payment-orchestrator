package middleware

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/platform/identity"
	"github.com/kuipid01/payconnect/internal/redis"
)

// Bucket identifies the scope a rate limit applies to.
type Bucket string

const (
	// BucketGlobal covers every request from a caller.
	BucketGlobal Bucket = "global"
	// BucketWrite covers mutating requests.
	BucketWrite Bucket = "write"
	// BucketPayment is the strictest bucket, applied to payment creation.
	BucketPayment Bucket = "payment"
	// BucketAuth covers every public authentication endpoint.
	BucketAuth Bucket = "auth"
	// BucketAuthStrict covers the credential endpoints (login, register,
	// password reset), where the threat is guessing rather than volume.
	BucketAuthStrict Bucket = "auth_strict"
)

// RateLimit installs a fixed-window limiter on the shared store.
//
// Keys are derived from the authenticated merchant or API key when available and
// the client IP otherwise, so one merchant cannot exhaust another's budget.
func RateLimit(store redis.Store, log *slog.Logger, bucket Bucket, limitPerMinute int, prefix string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if store == nil || limitPerMinute <= 0 {
			return c.Next()
		}
		key := fmt.Sprintf("%s:%s:%s:%d", prefix, bucket, callerSubject(c), time.Now().Unix()/60)
		count, err := store.Incr(c.UserContext(), key, 2*time.Minute)
		if err != nil {
			// A limiter that cannot reach its store must not take the API down.
			if log != nil {
				log.Warn("rate limit store unavailable", "error", err)
			}
			return c.Next()
		}
		if count > int64(limitPerMinute) {
			c.Set("Retry-After", "60")
			return apperrors.RateLimitExceeded()
		}
		return c.Next()
	}
}

// callerSubject prefers the authenticated principal over the caller IP.
func callerSubject(c *fiber.Ctx) string {
	if principal := identity.From(c); principal != nil && principal.MerchantID != "" {
		if principal.APIKeyID != "" {
			return "key_" + principal.APIKeyID
		}
		return "mch_" + principal.MerchantID
	}
	return "ip_" + c.IP()
}
