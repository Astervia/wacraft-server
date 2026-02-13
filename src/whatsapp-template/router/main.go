package whatsapp_template_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	whatsapp_template_handler "github.com/Astervia/wacraft-server/src/whatsapp-template/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/whatsapp-template")

	mainRoutes(group)
}

func mainRoutes(group fiber.Router) {
	group.Get("/", auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		whatsapp_template_handler.Get)
}
