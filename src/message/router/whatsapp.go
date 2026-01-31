package message_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	message_handler "github.com/Astervia/wacraft-server/src/message/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func whatsappRoutes(group fiber.Router) {
	wppGroup := group.Group("/whatsapp")

	wppGroup.Get("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageRead))
	wppGroup.Post("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageSend),
		message_handler.SendMessage)
	wppGroup.Get("/wam-id/:wamID",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageRead),
		message_handler.GetWamID)
	wppGroup.Post("/mark-as-read",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageSend),
		message_handler.MarkWhatsAppMessageAsReadToUser)
	wppGroup.Post("/send-typing",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyMessageSend),
		message_handler.SendTypingToUser)
}
