package providers

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the provider endpoints.
//
// Reading the catalogue is safe for any authenticated caller; every mutation of
// stored credentials (connect, rotate, delete, verify) requires a dashboard
// session, because an API key must never be able to change where money goes.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, h *Handler) {
	group := api.Group("/providers")
	group.Get("", h.Supported)
	group.Get("/connections", sessionOnly, h.ListConnections)
	group.Get("/connections/:id", sessionOnly, h.GetConnection)
	group.Post("/connections", sessionOnly, h.CreateConnection)
	group.Patch("/connections/:id", sessionOnly, h.UpdateConnection)
	group.Delete("/connections/:id", sessionOnly, h.DeleteConnection)
	group.Post("/connections/:id/test", sessionOnly, h.TestConnection)
	group.Post("/connections/:id/webhook-secret", sessionOnly, h.SetWebhookSecret)
}
