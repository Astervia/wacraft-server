package webhook_router

import (
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	webhook_handler "github.com/Astervia/wacraft-server/src/webhook/handler"
	"github.com/gofiber/fiber/v2"
)

func logRoutes(group fiber.Router) {
	logGroup := group.Group("/log")

	logGroup.Get("/",
		auth_middleware.UserMiddleware, auth_middleware.RoleMiddleware(user_model.Admin, user_model.Automation, user_model.Developer),
		webhook_handler.GetWebhookLogs)
	logGroup.Post("/send",
		auth_middleware.UserMiddleware, auth_middleware.RoleMiddleware(user_model.Admin, user_model.Automation, user_model.Developer),
		webhook_handler.GetWebhookLogs)
}
