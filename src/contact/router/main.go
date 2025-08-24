package contact_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	contact_handler "github.com/Astervia/wacraft-server/src/contact/handler"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/contact")

	mainRoutes(group)
}

func mainRoutes(group fiber.Router) {
	group.Get("/", auth_middleware.UserMiddleware, contact_handler.Get)
	group.Post("/", auth_middleware.UserMiddleware, contact_handler.CreateContact)
	group.Put("/", auth_middleware.UserMiddleware, contact_handler.UpdateContact)
	group.Delete("/", auth_middleware.UserMiddleware, contact_handler.DeleteContactByID) // Route for deleting by ID
}
