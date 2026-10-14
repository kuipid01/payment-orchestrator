package middleware

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/redis"
)

func newThrottle(maxFailures int, lockout time.Duration) *AuthThrottle {
	return NewAuthThrottle(redis.NewMemoryStore(), nil, maxFailures, lockout, "test")
}

func TestAuthThrottleBlocksAfterMaxFailures(t *testing.T) {
	ctx := context.Background()
	throttle := newThrottle(3, time.Minute)

	if throttle.Blocked(ctx, "ip_10.0.0.1") {
		t.Fatal("a fresh caller must not be blocked")
	}
	throttle.RecordFailure(ctx, "ip_10.0.0.1")
	throttle.RecordFailure(ctx, "ip_10.0.0.1")
	if throttle.Blocked(ctx, "ip_10.0.0.1") {
		t.Fatal("caller blocked before reaching the failure budget")
	}
	throttle.RecordFailure(ctx, "ip_10.0.0.1")
	if !throttle.Blocked(ctx, "ip_10.0.0.1") {
		t.Fatal("caller should be blocked once the budget is exhausted")
	}
	// The budget is per caller: a noisy address must not lock anyone else out.
	if throttle.Blocked(ctx, "ip_10.0.0.2") {
		t.Fatal("one caller's failures must not block another caller")
	}
}

func TestAuthThrottleResetClearsBudget(t *testing.T) {
	ctx := context.Background()
	throttle := newThrottle(1, time.Minute)

	throttle.RecordFailure(ctx, "ip_10.0.0.1")
	if !throttle.Blocked(ctx, "ip_10.0.0.1") {
		t.Fatal("expected the caller to be blocked after one failure")
	}
	throttle.Reset(ctx, "ip_10.0.0.1")
	if throttle.Blocked(ctx, "ip_10.0.0.1") {
		t.Fatal("a successful attempt must clear the failure budget")
	}
}

// A lockout is temporary: it must lift without operator action.
func TestAuthThrottleLockoutExpires(t *testing.T) {
	ctx := context.Background()
	throttle := newThrottle(1, 50*time.Millisecond)

	throttle.RecordFailure(ctx, "ip_10.0.0.1")
	if !throttle.Blocked(ctx, "ip_10.0.0.1") {
		t.Fatal("expected the caller to be blocked")
	}
	time.Sleep(90 * time.Millisecond)
	if throttle.Blocked(ctx, "ip_10.0.0.1") {
		t.Fatal("the lockout should have expired on its own")
	}
}

func TestAuthThrottleDisabled(t *testing.T) {
	ctx := context.Background()
	cases := map[string]*AuthThrottle{
		"no store":     NewAuthThrottle(nil, nil, 3, time.Minute, "test"),
		"no budget":    NewAuthThrottle(redis.NewMemoryStore(), nil, 0, time.Minute, "test"),
		"no lockout":   NewAuthThrottle(redis.NewMemoryStore(), nil, 3, 0, "test"),
		"nil throttle": nil,
		"negative max": NewAuthThrottle(redis.NewMemoryStore(), nil, -1, time.Minute, "test"),
	}
	for name, throttle := range cases {
		if throttle.Enabled() {
			t.Errorf("%s: throttle should be disabled", name)
		}
		if throttle.Blocked(ctx, "ip_10.0.0.1") {
			t.Errorf("%s: a disabled throttle must never block", name)
		}
	}
}

func TestCountsAsAuthFailure(t *testing.T) {
	if !CountsAsAuthFailure(apperrors.InvalidCredentials()) {
		t.Error("a rejected credential must consume the lockout budget")
	}
	if !CountsAsAuthFailure(apperrors.TokenInvalid("")) {
		t.Error("a rejected token must consume the lockout budget")
	}
	// A malformed request must not be able to lock an address out.
	if CountsAsAuthFailure(apperrors.Validation(apperrors.CodeInvalidRequest, "bad body")) {
		t.Error("a validation error must not consume the lockout budget")
	}
	if CountsAsAuthFailure(apperrors.NotFound("", "nope")) {
		t.Error("a not-found error must not consume the lockout budget")
	}
	if CountsAsAuthFailure(nil) {
		t.Error("nil is not a failure")
	}
	if CountsAsAuthFailure(errors.New("boom")) {
		t.Error("an untyped error must not consume the lockout budget")
	}
}

/* ------------------------------- integration -------------------------------- */

// throttleApp wires the throttle around a handler whose outcome the test
// controls. The ErrorHandler mirrors the real server so a guard that returns a
// typed error renders the same status.
func throttleApp(throttle *AuthThrottle, handler fiber.Handler, calls *int) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			appErr := apperrors.From(err)
			return c.Status(appErr.HTTPStatus).JSON(fiber.Map{"code": appErr.Code})
		},
	})
	app.Post("/auth/login", throttle.Handler(), func(c *fiber.Ctx) error {
		*calls++
		return handler(c)
	})
	return app
}

// renderRejection mimics how every handler in this codebase reports failure:
// apihttp.Responder writes the envelope and returns nil, so no error travels up
// the middleware chain and the throttle has to read the response status.
func renderRejection(code string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": code})
	}
}

func postLogin(t *testing.T, app *fiber.App) (int, string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodPost, "/auth/login", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode, string(body)
}

func TestThrottleHandlerLocksOutAfterFailures(t *testing.T) {
	throttle := newThrottle(2, time.Minute)
	calls := 0
	app := throttleApp(throttle, func(c *fiber.Ctx) error {
		return apperrors.InvalidCredentials()
	}, &calls)

	for i := 0; i < 2; i++ {
		status, _ := postLogin(t, app)
		if status != fiber.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, status)
		}
	}
	if calls != 2 {
		t.Fatalf("handler ran %d times, want 2 before the lockout", calls)
	}

	status, body := postLogin(t, app)
	if status != fiber.StatusTooManyRequests {
		t.Fatalf("locked-out status = %d, want 429 (body %s)", status, body)
	}
	// The lockout must short-circuit: the credential check never runs.
	if calls != 2 {
		t.Fatalf("handler ran %d times, want it skipped while locked out", calls)
	}
}

// This is the case that matters in production, and the one an error-only
// implementation silently misses: the handler renders 401 and returns nil.
func TestThrottleHandlerCountsRenderedRejections(t *testing.T) {
	throttle := newThrottle(2, time.Minute)
	calls := 0
	app := throttleApp(throttle, renderRejection(apperrors.CodeInvalidCredentials), &calls)

	for i := 0; i < 2; i++ {
		if status, body := postLogin(t, app); status != fiber.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401 (body %s)", i+1, status, body)
		}
	}
	status, body := postLogin(t, app)
	if status != fiber.StatusTooManyRequests {
		t.Fatalf("rendered 401s did not trip the lockout: status = %d, want 429 (body %s)", status, body)
	}
	if calls != 2 {
		t.Fatalf("handler ran %d times, want it skipped while locked out", calls)
	}
}

// A rendered validation failure must not consume the lockout budget.
func TestThrottleHandlerIgnoresRenderedValidationFailures(t *testing.T) {
	throttle := newThrottle(1, time.Minute)
	calls := 0
	app := throttleApp(throttle, func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": apperrors.CodeInvalidRequest})
	}, &calls)

	for i := 0; i < 4; i++ {
		if status, body := postLogin(t, app); status != fiber.StatusBadRequest {
			t.Fatalf("attempt %d: status = %d, want 400 (body %s)", i+1, status, body)
		}
	}
	if calls != 4 {
		t.Fatalf("handler ran %d times, want 4", calls)
	}
}

func TestThrottleHandlerClearsBudgetOnSuccess(t *testing.T) {
	throttle := newThrottle(2, time.Minute)
	calls := 0
	fail := true
	app := throttleApp(throttle, func(c *fiber.Ctx) error {
		if fail {
			return apperrors.InvalidCredentials()
		}
		return c.SendStatus(fiber.StatusOK)
	}, &calls)

	if status, _ := postLogin(t, app); status != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
	fail = false
	if status, _ := postLogin(t, app); status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	fail = true
	// The successful sign-in cleared the budget, so a single later failure must
	// not lock the caller out.
	if status, _ := postLogin(t, app); status != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 rather than a lockout", status)
	}
}

func TestThrottleHandlerIgnoresValidationFailures(t *testing.T) {
	throttle := newThrottle(1, time.Minute)
	calls := 0
	app := throttleApp(throttle, func(c *fiber.Ctx) error {
		return apperrors.Validation(apperrors.CodeInvalidRequest, "bad body")
	}, &calls)

	// Well past the budget of 1: if malformed requests counted as failures, the
	// third call would already be a 429 instead of a 400.
	for i := 0; i < 5; i++ {
		status, body := postLogin(t, app)
		if status != fiber.StatusBadRequest {
			t.Fatalf("attempt %d: status = %d, want 400 (body %s)", i+1, status, body)
		}
	}
	if calls != 5 {
		t.Fatalf("handler ran %d times, want 5: validation errors must not lock the caller out", calls)
	}
}
