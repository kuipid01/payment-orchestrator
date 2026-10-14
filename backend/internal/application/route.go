package application

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts the application endpoints.
func RegisterRoutes(api fiber.Router, sessionOnly fiber.Handler, handler *Handler) {
	group := api.Group("/applications")
	group.Get("", handler.List)
	group.Get("/:id", handler.Get)
	group.Post("", sessionOnly, handler.Create)
	group.Patch("/:id", sessionOnly, handler.Update)
	group.Delete("/:id", sessionOnly, handler.Delete)
}
