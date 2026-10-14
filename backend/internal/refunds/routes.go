package refunds

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the refund endpoints.
//
// Creating a refund requires a developer API key (money movement must be
// attributable to a scoped credential); reads accept either credential and stay
// environment-scoped for API keys.
func RegisterRoutes(api fiber.Router, h *Handler) {
	api.Post("/payments/:id/refund", h.Create)
	api.Get("/refunds", h.List)
	api.Get("/refunds/:id", h.Get)
}
