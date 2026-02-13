package status_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	status_handler "github.com/Astervia/wacraft-server/src/status/handler"
	"github.com/gofiber/fiber/v2"
)

func whatsappRoutes(group fiber.Router) {
	wppGroup := group.Group("/whatsapp")

	wppGroup.Get("/wam-id/:wamID", auth_middleware.UserMiddleware, auth_middleware.EmailVerifiedMiddleware, billing_middleware.ThroughputMiddleware, status_handler.GetWamID)
}
