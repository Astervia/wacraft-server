package webhook_router

import (
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	webhook_handler "github.com/Astervia/wacraft-server/src/webhook/handler"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/webhook")

	mainRoutes(group)
	logRoutes(group)
}

func mainRoutes(group fiber.Router) {
	group.Get("/",
		auth_middleware.UserMiddleware, auth_middleware.RoleMiddleware(user_model.Admin, user_model.Automation, user_model.Developer),
		webhook_handler.GetWebhooks)
	group.Post("/",
		auth_middleware.UserMiddleware, auth_middleware.RoleMiddleware(user_model.Admin, user_model.Automation, user_model.Developer),
		webhook_handler.CreateWebhook)
	group.Put("/",
		auth_middleware.UserMiddleware, auth_middleware.RoleMiddleware(user_model.Admin, user_model.Automation, user_model.Developer),
		webhook_handler.UpdateWebhook)
	group.Delete("/",
		auth_middleware.UserMiddleware, auth_middleware.RoleMiddleware(user_model.Admin, user_model.Automation, user_model.Developer),
		webhook_handler.DeleteWebhookByID)
	group.Get("/content/:keyName/like/:likeText",
		auth_middleware.UserMiddleware, auth_middleware.RoleMiddleware(user_model.Admin, user_model.Automation, user_model.Developer),
		webhook_handler.ContentKeyLike)
}
