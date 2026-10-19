package api_keys

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the API key endpoints. Every one requires a dashboard
// session; key management also requires an owner or admin, and an API key must
// never be able to mint or revoke another key.
func RegisterRoutes(api fiber.Router, sessionOnly, applicationManager fiber.Handler, h *Handler) {
	group := api.Group("/api-keys")
	group.Post("", sessionOnly, applicationManager, h.Create)
	group.Get("", sessionOnly, h.List)
	group.Post("/:id/rotate", sessionOnly, applicationManager, h.Rotate)
	group.Post("/:id/reveal", sessionOnly, h.Reveal)
	group.Delete("/:id/purge", sessionOnly, applicationManager, h.DeleteRevoked)
	group.Delete("/:id", sessionOnly, applicationManager, h.Revoke)
}
