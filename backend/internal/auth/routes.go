package auth

import "github.com/gofiber/fiber/v2"

// PublicGuards carries the middleware applied to the unauthenticated auth
// endpoints. Each field is optional (nil means "no guard"), so the routes stay
// usable in tests and in wiring that has no limiter.
type PublicGuards struct {
	// General caps every auth endpoint.
	General fiber.Handler
	// Strict applies the tighter budget to the credential endpoints, where the
	// threat is guessing rather than request volume.
	Strict fiber.Handler
	// Lockout refuses a caller that keeps presenting rejected credentials.
	Lockout fiber.Handler
}

// RegisterPublicRoutes mounts unauthenticated auth endpoints.
//
// Registration, login and the password-reset pair carry the stricter budget:
// they are the endpoints an attacker hammers with candidate credentials. The
// refresh/logout/verification endpoints only do work with a token the caller
// already holds, so they get the general budget.
func RegisterPublicRoutes(v1 fiber.Router, guards PublicGuards, h *Handler) {
	if guards.General == nil {
		guards.General = allowAll()
	}
	if guards.Strict == nil {
		guards.Strict = allowAll()
	}
	if guards.Lockout == nil {
		guards.Lockout = allowAll()
	}

	auth := v1.Group("/auth", guards.General, guards.Lockout)
	auth.Post("/register", guards.Strict, h.Register)
	auth.Post("/login", guards.Strict, h.Login)
	auth.Post("/google", guards.Strict, h.Google)
	auth.Post("/forgot-password", guards.Strict, h.ForgotPassword)
	auth.Post("/reset-password", guards.Strict, h.ResetPassword)
	auth.Post("/refresh", h.Refresh)
	auth.Post("/logout", h.Logout)
	auth.Post("/verify-email", h.VerifyEmail)
	auth.Post("/resend-verification", h.ResendVerification)
}

// RegisterAuthenticatedRoutes mounts endpoints that need a bearer token.
func RegisterAuthenticatedRoutes(api fiber.Router, h *Handler) {
	api.Get("/auth/me", h.Me)
	// The organization switcher: list every membership, then move the session to
	// one of them. Both are session-resolution endpoints, so they sit alongside
	// /auth/me rather than behind an additional guard.
	api.Get("/auth/memberships", h.Memberships)
	api.Post("/auth/switch", h.Switch)
}

func allowAll() fiber.Handler {
	return func(c *fiber.Ctx) error { return c.Next() }
}
