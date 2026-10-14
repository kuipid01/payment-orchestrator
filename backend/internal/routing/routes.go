package routing

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the routing endpoints. Reading the routing overview is
// safe for any authenticated caller; changing where money is routed requires a
// dashboard session.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, h *Handler) {
	group := api.Group("/routing")
	group.Get("", h.Get)
	group.Patch("", sessionOnly, h.Update)
	group.Get("/rules", h.ListRules)
	group.Post("/rules", sessionOnly, h.CreateRule)
	group.Get("/rules/:id", h.GetRule)
	group.Patch("/rules/:id", sessionOnly, h.UpdateRule)
	group.Delete("/rules/:id", sessionOnly, h.DeleteRule)
}
