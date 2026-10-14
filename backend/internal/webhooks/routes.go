package webhooks

import "github.com/gofiber/fiber/v2"

// RegisterReceiverRoutes mounts the public provider webhook receivers. They are
// unauthenticated by design: authenticity comes from the provider's signature.
func RegisterReceiverRoutes(v1 fiber.Router, h *Handler) {
	v1.Post("/webhooks/:provider/:connection_id", h.Receive)
}

// RegisterRoutes mounts the merchant-facing webhook management endpoints.
// Reading deliveries is safe for any authenticated credential; configuring
// endpoints and forcing retries requires a dashboard session.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, h *Handler) {
	group := api.Group("/webhook-endpoints")
	group.Get("", h.ListEndpoints)
	group.Get("/:id", h.GetEndpoint)
	group.Post("", sessionOnly, h.CreateEndpoint)
	// A sample delivery writes a delivery row and signs a request with the
	// endpoint's secret, so it is a session action like the rest, not a read.
	group.Post("/:id/test", sessionOnly, h.TestEndpoint)
	group.Patch("/:id", sessionOnly, h.UpdateEndpoint)
	group.Delete("/:id", sessionOnly, h.DeleteEndpoint)

	inbound := api.Group("/webhook-events")
	inbound.Get("", h.ListInboundEvents)

	deliveries := api.Group("/webhook-deliveries")
	deliveries.Get("", h.ListDeliveries)
	deliveries.Get("/:id", h.GetDelivery)
	deliveries.Post("/:id/retry", sessionOnly, h.RetryDelivery)
}
