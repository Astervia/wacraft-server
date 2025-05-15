package user_router

import (
	user_handler "github.com/Astervia/wacraft-server/src/user/handler"
	"github.com/gofiber/fiber/v2"
)

func authRoutes(group fiber.Router) {
	oauthGroup := group.Group("/oauth")
	oauthGroup.Post("/token", user_handler.OAuthTokenHandler)
}
