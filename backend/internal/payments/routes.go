package payments

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the payment endpoints.
//
// Payment creation is the one place where the environment of the credential is
// authoritative, so it is API-key-only. Reads accept either credential and are
// still environment-scoped at the handler.
//
// createGuard is the stricter, payment-specific rate limiter: creating a charge
// is far more expensive than reading one, so it gets its own budget. A nil guard
// means "no extra limiter".
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, createGuard fiber.Handler, h *Handler) {
	if createGuard == nil {
		createGuard = func(c *fiber.Ctx) error { return c.Next() }
	}
	if sessionOnly == nil {
		sessionOnly = func(c *fiber.Ctx) error { return c.Next() }
	}
	api.Post("/payments", createGuard, h.Create)
	// A test transaction spends sandbox money against a stored connection on the
	// merchant's behalf, so it is dashboard-only: an API key must never reach it.
	api.Post("/payments/test", sessionOnly, h.Test)
	api.Get("/payments", h.List)
	api.Get("/payments/verify", h.Verify)
	api.Get("/payments/:id", h.Get)
	api.Get("/payments/:id/verify", h.Verify)
	api.Get("/dashboard/metrics", h.Metrics)
}
