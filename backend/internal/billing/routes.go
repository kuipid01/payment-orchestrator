package billing

import "github.com/gofiber/fiber/v2"

// RegisterPublicRoutes mounts the price list. It must be registered before the
// authenticated group installs its prefix handler, or Fiber would demand a
// session for a page that exists to be read by nobody in particular.
//
// The guard is a rate limit only: the catalog is public by design, and a public
// page that can be hammered for free is exactly what a limiter is for.
func RegisterPublicRoutes(v1 fiber.Router, guard fiber.Handler, h *Handler) {
	group := v1.Group("/plans")
	if guard != nil {
		group.Get("", guard, h.Catalog)
		return
	}
	group.Get("", h.Catalog)
}

// RegisterRoutes mounts the subscription surface. Reading is safe for any
// authenticated caller; changing a plan requires a dashboard session, because a
// plan is a billing fact and an API key is not a person.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, h *Handler) {
	group := api.Group("/billing")
	group.Get("/plan", h.Plan)
	group.Get("/usage", h.Usage)
	group.Get("/events", h.Events)
	group.Post("/checkout", sessionOnly, h.Checkout)
	group.Post("/trial", sessionOnly, h.StartTrial)
	group.Post("/cancel", sessionOnly, h.Cancel)
	group.Post("/resume", sessionOnly, h.Resume)
}
