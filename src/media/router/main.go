package media_router

import (
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/media")

	mainRoutes(group)
	whatsappRoutes(group)
}

func mainRoutes(group fiber.Router) {
}
