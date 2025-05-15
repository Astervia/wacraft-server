package message_websocket

import (
	message_handler "github.com/Astervia/wacraft-server/src/message/handler"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Route(app fiber.Router) {
	group := app.Group("/message")

	// This route must handle the registering, broadcasting, and unregistering of the connections.
	group.Get(
		"/new",
		websocket.New(message_handler.NewMessageSubscription),
	)
}
