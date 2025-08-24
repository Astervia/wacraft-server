package message_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	message_handler "github.com/Astervia/wacraft-server/src/message/handler"
	"github.com/gofiber/fiber/v2"
)

func conversationRoutes(group fiber.Router) {
	convGroup := group.Group("/conversation")

	convGroup.Get("", auth_middleware.UserMiddleware, message_handler.GetConversations)

	convGroup.Get("/count", auth_middleware.UserMiddleware, message_handler.CountDistinctConversations)

	convGroup.Get("/messaging-product-contact/:messagingProductContactID",
		auth_middleware.UserMiddleware, message_handler.GetConversation)

	convGroup.Get("/count/messaging-product-contact/:messagingProductContactID",
		auth_middleware.UserMiddleware, message_handler.CountConversationsByMessagingProductContact)

	convGroup.Get("/messaging-product-contact/:messagingProductContactID/content/like/:likeText",
		auth_middleware.UserMiddleware, message_handler.ConversationContentLikeByMessagingProductContact)

	convGroup.Get("/count/messaging-product-contact/:messagingProductContactID/content/like/:likeText",
		auth_middleware.UserMiddleware, message_handler.CountConversationContentLike)
}
