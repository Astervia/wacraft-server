package status_websocket

import (
	status_handler "github.com/Astervia/wacraft-server/src/status/handler"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Route(app fiber.Router) {
	group := app.Group("/status")

	// This route must handle the registering, broadcasting, and unregistering of the connections.
	group.Get(
		"/new",
		websocket.New(status_handler.NewStatusSubscription),
	)
}
