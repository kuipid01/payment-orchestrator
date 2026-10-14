package logs

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the log endpoints. Request logs follow the caller's
// credential and environment; the audit trail is dashboard-only.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, h *Handler) {
	api.Get("/logs", h.List)
	api.Get("/logs/:id", h.Get)
	api.Get("/audit-logs", sessionOnly, h.Audit)
}
