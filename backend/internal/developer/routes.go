package developer

import "github.com/gofiber/fiber/v2"

// PublicGuards carries the middleware applied to the unauthenticated invitation
// endpoints. Each field is optional (nil means "no guard") so the routes stay
// usable in tests and in wiring that has no limiter.
//
// Reading an invitation and accepting one are credential-grade operations: the
// accept endpoint verifies a password and, for a new address, creates an account.
// They therefore take the same budget as login rather than the general one.
type PublicGuards struct {
	// General caps every invitation endpoint.
	General fiber.Handler
	// Strict applies the tighter budget to the submit, which verifies a
	// credential and can create an account.
	Strict fiber.Handler
	// Lockout refuses a caller that keeps presenting rejected credentials.
	Lockout fiber.Handler
}

// RegisterPublicRoutes mounts the invitation endpoints on the public router. The
// invitee has no session yet — that is the whole point of the flow — so these
// must be registered before anything installs a `Use` handler under /v1.
//
// This split mirrors auth.RegisterPublicRoutes / RegisterAuthenticatedRoutes for
// that reason: Fiber applies a prefix `Use` handler to routes registered after
// it, so mounting a public route inside the authenticated section would silently
// require authentication on it.
func RegisterPublicRoutes(v1 fiber.Router, guards PublicGuards, handler *Handler) {
	if guards.General == nil {
		guards.General = allowAll()
	}
	if guards.Strict == nil {
		guards.Strict = allowAll()
	}
	if guards.Lockout == nil {
		guards.Lockout = allowAll()
	}

	invitations := v1.Group("/developers/invitations", guards.General, guards.Lockout)
	invitations.Get("/:token", handler.PreviewInvitation)
	invitations.Post("/:token/accept", guards.Strict, handler.AcceptInvitation)
}

// RegisterRoutes mounts the session-only developer endpoints.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, handler *Handler) {
	group := api.Group("/developers", sessionOnly)
	group.Get("", handler.List)
	group.Get("/me", handler.MyAccess)
	group.Post("/me/applications/:id/test-webhook", handler.TestWebhook)
	group.Post("/invite", handler.Invite)
	group.Post("/accept", handler.Accept) // deprecated; see the handler
	group.Delete("/invitations", handler.Revoke)
}

func allowAll() fiber.Handler {
	return func(c *fiber.Ctx) error { return c.Next() }
}
