package messaging_product_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	messaging_product_handler "github.com/Astervia/wacraft-server/src/messaging-product/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/messaging-product")

	mainRoutes(group)
	contactRoutes(group)
}

func mainRoutes(group fiber.Router) {
	group.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		messaging_product_handler.Get)
}
