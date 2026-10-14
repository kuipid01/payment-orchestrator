package api_keys

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the API key endpoints. Every one requires a dashboard
// session: minting credentials is merchant administration, and an API key must
// never be able to mint or revoke another key.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, h *Handler) {
	group := api.Group("/api-keys")
	group.Post("", sessionOnly, h.Create)
	group.Get("", sessionOnly, h.List)
	group.Post("/:id/rotate", sessionOnly, h.Rotate)
	group.Post("/:id/reveal", sessionOnly, h.Reveal)
	group.Delete("/:id/purge", sessionOnly, h.DeleteRevoked)
	group.Delete("/:id", sessionOnly, h.Revoke)
}
