package campaign_websocket

import (
	campaign_handler "github.com/Astervia/wacraft-server/src/campaign/handler"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Route(app fiber.Router) {
	group := app.Group("/campaign")

	// This route must handle the registering, broadcasting, and unregistering of the connections.
	group.Get(
		"/whatsapp/send/:campaignID",
		websocket.New(campaign_handler.SendWhatsAppCampaignSubscription),
	)
}
