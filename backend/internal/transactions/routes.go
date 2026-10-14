package transactions

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the ledger endpoints. Reads accept either credential and
// stay environment-scoped for API keys.
func RegisterRoutes(api fiber.Router, h *Handler) {
	api.Get("/transactions", h.List)
	api.Get("/transactions/:id", h.Get)
}
