package middleware

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/redis"
)

// AuthThrottle temporarily locks out a caller that keeps presenting rejected
// credentials.
//
// It complements the fixed-window limiter rather than duplicating it: the
// limiter caps request *volume*, which throttles a fast guesser but still lets a
// slow one keep trying forever. This counts failed *attempts* so that an
// address which has burned through its budget is refused outright until the
// lockout expires on its own.
//
// The failure signal is the error the handler already returns, so no handler
// needs to know the throttle exists.
type AuthThrottle struct {
	Store       redis.Store
	Logger      *slog.Logger
	MaxFailures int
	Lockout     time.Duration
	Prefix      string
}

// NewAuthThrottle builds a throttle. A nil store, or a non-positive failure
// budget, disables it.
func NewAuthThrottle(store redis.Store, logger *slog.Logger, maxFailures int, lockout time.Duration, prefix string) *AuthThrottle {
	if prefix == "" {
		prefix = "authfail"
	}
	return &AuthThrottle{
		Store:       store,
		Logger:      logger,
		MaxFailures: maxFailures,
		Lockout:     lockout,
		Prefix:      prefix,
	}
}

// Enabled reports whether the throttle will act.
func (t *AuthThrottle) Enabled() bool {
	return t != nil && t.Store != nil && t.MaxFailures > 0 && t.Lockout > 0
}

// AuthSubject identifies the caller being throttled. The public auth endpoints
// are unauthenticated, so the client IP is the only identity available; it
// resolves through X-Forwarded-For only when a trusted proxy is configured.
func AuthSubject(c *fiber.Ctx) string { return "ip_" + c.IP() }

// Blocked reports whether the subject has exhausted its failure budget. The
// counter expires with the lockout window, so the block lifts by itself.
func (t *AuthThrottle) Blocked(ctx context.Context, subject string) bool {
	if !t.Enabled() {
		return false
	}
	value, ok, err := t.Store.Get(ctx, t.key(subject))
	if err != nil {
		// A throttle that cannot read its store must not lock the API out.
		t.warn("auth throttle store unavailable", err)
		return false
	}
	if !ok {
		return false
	}
	failures, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return failures >= t.MaxFailures
}

// RecordFailure counts one rejected attempt.
func (t *AuthThrottle) RecordFailure(ctx context.Context, subject string) {
	if !t.Enabled() {
		return
	}
	if _, err := t.Store.Incr(ctx, t.key(subject), t.Lockout); err != nil {
		t.warn("auth throttle store unavailable", err)
	}
}

// Reset clears the failure budget after a successful attempt.
func (t *AuthThrottle) Reset(ctx context.Context, subject string) {
	if !t.Enabled() {
		return
	}
	if err := t.Store.Del(ctx, t.key(subject)); err != nil {
		t.warn("auth throttle store unavailable", err)
	}
}

// Handler refuses a locked-out caller, then counts or clears the attempt
// according to the handler's outcome.
//
// The outcome is read from two signals on purpose. Handlers render their own
// error envelope through apihttp.Responder and return nil, so a rejected
// credential leaves no error on the chain — only a 401 on the response. A
// middleware that returns a typed error instead (the rate limiter, for one)
// never commits a status. Reading both keeps the throttle correct either way.
func (t *AuthThrottle) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !t.Enabled() {
			return c.Next()
		}
		subject := AuthSubject(c)
		if t.Blocked(c.UserContext(), subject) {
			c.Set("Retry-After", strconv.Itoa(int(t.Lockout.Seconds())))
			return apperrors.RateLimitExceeded().
				WithMessage("Too many failed attempts from this address. Try again later.")
		}
		err := c.Next()
		switch {
		case err != nil && CountsAsAuthFailure(err):
			t.RecordFailure(c.UserContext(), subject)
		case statusRejectedCredential(c):
			t.RecordFailure(c.UserContext(), subject)
		case err == nil && c.Response().StatusCode() < fiber.StatusBadRequest:
			t.Reset(c.UserContext(), subject)
		}
		return err
	}
}

// statusRejectedCredential reports whether the rendered response is a 401. Any
// other failure status (400 validation, 403 unverified, 429 throttled) neither
// consumes the lockout budget nor clears it.
func statusRejectedCredential(c *fiber.Ctx) bool {
	return c.Response().StatusCode() == fiber.StatusUnauthorized
}

// CountsAsAuthFailure reports whether an error means a credential was rejected,
// as opposed to a malformed or unauthorized request. Only guessed-credential
// failures should consume the lockout budget: a client sending bad JSON must not
// be able to lock an address out.
func CountsAsAuthFailure(err error) bool {
	return apperrors.Is(err, apperrors.CodeInvalidCredentials) ||
		apperrors.Is(err, apperrors.CodeTokenInvalid)
}

func (t *AuthThrottle) key(subject string) string { return t.Prefix + ":" + subject }

func (t *AuthThrottle) warn(message string, err error) {
	if t.Logger != nil {
		t.Logger.Warn(message, "error", err)
	}
}
