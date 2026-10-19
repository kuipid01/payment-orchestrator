package application

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the application endpoints.
func RegisterRoutes(api fiber.Router, sessionOnly, applicationManager fiber.Handler, handler *Handler) {
	group := api.Group("/applications")
	group.Get("", handler.List)
	group.Get("/:id", handler.Get)
	group.Post("", sessionOnly, applicationManager, handler.Create)
	group.Patch("/:id", sessionOnly, applicationManager, handler.Update)
	group.Delete("/:id", sessionOnly, applicationManager, handler.Delete)
}
