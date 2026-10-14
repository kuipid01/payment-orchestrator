package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// The guards are the only protection on the unauthenticated auth endpoints, so
// a wiring mistake is a security bug. These tests pin which guard runs on which
// route.
//
// Guards pass through by calling c.Next(); the guard at the stage under test
// terminates the chain so the real handlers never run and no service is needed.
type guardRecorder struct {
	general, strict, lockout int
}

func (g *guardRecorder) pass(n *int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		*n++
		return c.Next()
	}
}

func (g *guardRecorder) stop(n *int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		*n++
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func (g *guardRecorder) app(terminateAt string) *fiber.App {
	var guards PublicGuards
	switch terminateAt {
	case "strict":
		guards = PublicGuards{
			General: g.pass(&g.general),
			Lockout: g.pass(&g.lockout),
			Strict:  g.stop(&g.strict),
		}
	default: // "lockout"
		guards = PublicGuards{
			General: g.pass(&g.general),
			Lockout: g.stop(&g.lockout),
			Strict:  g.pass(&g.strict),
		}
	}
	app := fiber.New()
	RegisterPublicRoutes(app.Group("/v1"), guards, &Handler{})
	return app
}

func post(t *testing.T, app *fiber.App, path string) int {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodPost, path, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s: request failed: %v", path, err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// The credential endpoints carry all three guards, and the group guards must run
// before the route guard.
func TestPublicRoutesStrictGuardCoverage(t *testing.T) {
	strictRoutes := []string{
		"/v1/auth/register",
		"/v1/auth/login",
		"/v1/auth/google",
		"/v1/auth/forgot-password",
		"/v1/auth/reset-password",
	}
	for _, path := range strictRoutes {
		recorder := &guardRecorder{}
		app := recorder.app("strict")
		if status := post(t, app, path); status != fiber.StatusNoContent {
			t.Fatalf("%s: status = %d, want 204 (a guard should have stopped it)", path, status)
		}
		if recorder.general != 1 || recorder.lockout != 1 || recorder.strict != 1 {
			t.Fatalf("%s: guard runs general=%d lockout=%d strict=%d, want 1/1/1",
				path, recorder.general, recorder.lockout, recorder.strict)
		}
	}
}

// Every public auth route must pass through the lockout, including the ones
// without the strict budget.
func TestPublicRoutesAllCoveredByLockout(t *testing.T) {
	paths := []string{
		"/v1/auth/register", "/v1/auth/login", "/v1/auth/google",
		"/v1/auth/refresh", "/v1/auth/logout", "/v1/auth/verify-email",
		"/v1/auth/resend-verification", "/v1/auth/forgot-password",
		"/v1/auth/reset-password",
	}
	for _, path := range paths {
		recorder := &guardRecorder{}
		app := recorder.app("lockout")
		if status := post(t, app, path); status != fiber.StatusNoContent {
			t.Fatalf("%s: status = %d, want 204", path, status)
		}
		if recorder.general != 1 || recorder.lockout != 1 {
			t.Fatalf("%s: general=%d lockout=%d, want the lockout to run on every route",
				path, recorder.general, recorder.lockout)
		}
	}
}

// NOTE: the absence of the strict guard on the token endpoints (/auth/refresh,
// /auth/logout, /auth/verify-email, /auth/resend-verification) is not asserted
// here: the terminating lockout guard sits before the route-level guard, so a
// guard-count assertion could not distinguish "absent" from "unreached". It is
// verified against the running server instead, where a strict budget below the
// request count must throttle login while refresh keeps working.
