package webhook_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	webhook_handler "github.com/Astervia/wacraft-server/src/webhook/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func logRoutes(group fiber.Router) {
	logGroup := group.Group("/log")

	logGroup.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWebhookRead),
		billing_middleware.ThroughputMiddleware,
		webhook_handler.GetWebhookLogs)
	logGroup.Post("/send",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWebhookManage),
		billing_middleware.ThroughputMiddleware,
		webhook_handler.GetWebhookLogs)
}
