package message_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	message_handler "github.com/Astervia/wacraft-server/src/message/handler"
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
		auth_middleware.UserMiddleware, message_handler.Get)

	group.Get("/count",
		auth_middleware.UserMiddleware, message_handler.Count)

	group.Get("/content/like/:likeText",
		auth_middleware.UserMiddleware, message_handler.ContentLike)

	group.Get("/count/content/like/:likeText",
		auth_middleware.UserMiddleware, message_handler.CountContentLike)

	group.Get("/content/:keyName/like/:likeText",
		auth_middleware.UserMiddleware, message_handler.ContentKeyLike)
}
