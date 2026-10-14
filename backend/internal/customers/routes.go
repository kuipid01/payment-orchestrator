package customers

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the customer endpoints. Reads accept either credential
// (scoped to the caller's merchant); writes require a dashboard session.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, h *Handler) {
	group := api.Group("/customers")
	group.Get("", h.List)
	group.Get("/:id", h.Get)
	group.Post("", h.Create)
	group.Patch("/:id", sessionOnly, h.Update)
	group.Delete("/:id", sessionOnly, h.Delete)
}
