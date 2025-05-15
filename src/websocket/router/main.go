package websocket_router

import (
	auth_websocket_middleware "github.com/Astervia/wacraft-server/src/auth/middleware/websocket"
	"github.com/gofiber/fiber/v2"
)

// Register websocket authentication routes and group in a single app that must be passed to all other websocket routers
func Route(app *fiber.App) fiber.Router {
	// Group all websocket routes
	group := app.Group("/websocket")
	// Authenticate websocket simple connections
	group.Use(
		"",
		auth_websocket_middleware.UserMiddleware,
	)

	return group
}
