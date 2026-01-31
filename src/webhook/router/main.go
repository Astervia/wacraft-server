package webhook_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	webhook_handler "github.com/Astervia/wacraft-server/src/webhook/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/webhook")

	mainRoutes(group)
	logRoutes(group)
}

func mainRoutes(group fiber.Router) {
	group.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWebhookRead),
		webhook_handler.GetWebhooks)
	group.Post("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWebhookManage),
		webhook_handler.CreateWebhook)
	group.Put("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWebhookManage),
		webhook_handler.UpdateWebhook)
	group.Delete("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWebhookManage),
		webhook_handler.DeleteWebhookByID)
	group.Get("/content/:keyName/like/:likeText",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWebhookRead),
		webhook_handler.ContentKeyLike)
}
