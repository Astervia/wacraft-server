package message_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	message_handler "github.com/Astervia/wacraft-server/src/message/handler"
	"github.com/gofiber/fiber/v2"
)

func whatsappRoutes(group fiber.Router) {
	wppGroup := group.Group("/whatsapp")

	wppGroup.Get("", auth_middleware.UserMiddleware)
	wppGroup.Post("", auth_middleware.UserMiddleware, message_handler.SendMessage)
	wppGroup.Get("/wam-id/:wamId", auth_middleware.UserMiddleware, message_handler.GetWamId)
	wppGroup.Post("/mark-as-read", auth_middleware.UserMiddleware, message_handler.MarkWhatsAppMessageAsReadToUser)
}
