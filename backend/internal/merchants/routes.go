package merchants

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the merchant profile endpoints. Both require a
// dashboard session: profile changes are merchant administration.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, h *Handler) {
	group := api.Group("/merchant")
	group.Get("", sessionOnly, h.Get)
	group.Patch("", sessionOnly, h.Update)
}
