package status_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	status_handler "github.com/Astervia/wacraft-server/src/status/handler"
	"github.com/gofiber/fiber/v2"
)

func whatsappRoutes(group fiber.Router) {
	wppGroup := group.Group("/whatsapp")

	wppGroup.Get("/wam-id/:wamId", auth_middleware.UserMiddleware, status_handler.GetWamId)
}
