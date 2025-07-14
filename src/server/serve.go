package server

import (
	"fmt"

	// PREMIUM STARTS
	campaign_router "github.com/Astervia/wacraft-server/src/campaign/router"
	campaign_websocket "github.com/Astervia/wacraft-server/src/campaign/websocket-router"
	// PREMIUM ENDS
	"github.com/Astervia/wacraft-server/src/config/env"
	contact_router "github.com/Astervia/wacraft-server/src/contact/router"
	media_router "github.com/Astervia/wacraft-server/src/media/router"
	message_router "github.com/Astervia/wacraft-server/src/message/router"
	message_websocket "github.com/Astervia/wacraft-server/src/message/websocket-router"
	messaging_product_router "github.com/Astervia/wacraft-server/src/messaging-product/router"
	status_router "github.com/Astervia/wacraft-server/src/status/router"
	status_websocket "github.com/Astervia/wacraft-server/src/status/websocket-router"
	user_router "github.com/Astervia/wacraft-server/src/user/router"
	"github.com/Astervia/wacraft-server/src/validators"
	webhook_config "github.com/Astervia/wacraft-server/src/webhook-in/config"
	webhook_router "github.com/Astervia/wacraft-server/src/webhook/router"
	"github.com/Astervia/wacraft-server/src/websocket"
	whatsapp_template_router "github.com/Astervia/wacraft-server/src/whatsapp-template/router"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/pterm/pterm"
)

func serve() {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))

	validators.InitValidators()

	// Serving http endpoints
	webhook_config.ServeWebhook(app)
	makeDocs(app)
	user_router.Route(app)
	contact_router.Route(app)
	messaging_product_router.Route(app)
	message_router.Route(app)
	// PREMIUM STARTS
	campaign_router.Route(app)
	// PREMIUM ENDS
	media_router.Route(app)
	webhook_router.Route(app)
	whatsapp_template_router.Route(app)
	status_router.Route(app)

	// Serving websockets
	websocketRouter := websocket.Main(app)
	message_websocket.Route(websocketRouter)
	// PREMIUM STARTS
	campaign_websocket.Route(websocketRouter)
	// PREMIUM ENDS
	status_websocket.Route(websocketRouter)

	err := app.Listen(fmt.Sprintf(":%s", env.ServerPort))
	pterm.DefaultLogger.Fatal(
		fmt.Sprintf("%v", err),
	)
}
