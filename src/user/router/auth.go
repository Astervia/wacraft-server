package user_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	user_handler "github.com/Astervia/wacraft-server/src/user/handler"
	"github.com/gofiber/fiber/v2"
)

func authRoutes(group fiber.Router) {
	oauthGroup := group.Group("/oauth")
	oauthGroup.Post("/token",
		auth_middleware.LoginRateLimiter,
		user_handler.OAuthTokenHandler,
	)
}
