package message_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	message_handler "github.com/Astervia/wacraft-server/src/message/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/message")

	mainRoutes(group)
	whatsappRoutes(group)
	conversationRoutes(group)
}

func mainRoutes(group fiber.Router) {
	group.Get("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageRead),
		billing_middleware.ThroughputMiddleware,
		message_handler.Get)

	group.Get("/count",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageRead),
		billing_middleware.ThroughputMiddleware,
		message_handler.Count)

	group.Get("/content/like/:likeText",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageRead),
		billing_middleware.ThroughputMiddleware,
		message_handler.ContentLike)

	group.Get("/count/content/like/:likeText",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageRead),
		billing_middleware.ThroughputMiddleware,
		message_handler.CountContentLike)

	group.Get("/content/:keyName/like/:likeText",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageRead),
		billing_middleware.ThroughputMiddleware,
		message_handler.ContentKeyLike)
}
