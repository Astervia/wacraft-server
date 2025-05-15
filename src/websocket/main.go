package websocket

import (
	websocket_router "github.com/Astervia/wacraft-server/src/websocket/router"
	"github.com/gofiber/fiber/v2"
	"github.com/pterm/pterm"
)

func Main(app *fiber.App) fiber.Router {
	pterm.DefaultLogger.Info("Loading websocket server initial configuration...")
	pterm.DefaultLogger.Info("Registering middleware for all websocket routes...")

	websocketRouter := websocket_router.Route(app)

	pterm.DefaultLogger.Info("Websocket initial configuration loaded")
	return websocketRouter
}
