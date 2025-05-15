package webhook_config

import (
	"github.com/Astervia/wacraft-server/src/config/env"
	webhook_handler "github.com/Astervia/wacraft-server/src/webhook-in/handler"
	wh_model "github.com/Rfluid/whatsapp-cloud-api/src/webhook/model"
	auth_middleware "github.com/Rfluid/whatsapp-webhook-server/src/auth/middleware"
	server_service "github.com/Rfluid/whatsapp-webhook-server/src/server/service"
	webhook_model "github.com/Rfluid/whatsapp-webhook-server/src/webhook/model"
	webhook_service "github.com/Rfluid/whatsapp-webhook-server/src/webhook/service"
	"github.com/gofiber/fiber/v2"
)

var Hook = webhook_service.Config{
	Path: "",
	ChangeHandlers: []webhook_model.ChangeHandler{
		webhook_handler.MessageHandler,
	},
	CtxHandler: defaultCtxHandler,
	PostMiddlewares: [](func(ctx *fiber.Ctx) error){
		func(ctx *fiber.Ctx) error {
			appSecret := env.MetaAppSecret
			if appSecret == "" {
				return ctx.Next()
			}
			return auth_middleware.VerifyMetaSignature(appSecret)(ctx)
		},
	},
	GetMiddlewares: [](func(ctx *fiber.Ctx) error){
		func(ctx *fiber.Ctx) error {
			metaVerifyToken := env.MetaVerifyToken
			if metaVerifyToken == "" {
				return ctx.Next()
			}
			return auth_middleware.MetaVerificationRequestToken(metaVerifyToken)(ctx)
		},
	},
}

func ServeWebhook(app *fiber.App) {
	server := server_service.NewConfig(app, "/webhook-in")
	server_service.Bootstrap(server, &Hook)
}

func defaultCtxHandler(ctx *fiber.Ctx, body *wh_model.WebhookBody) error {
	// return webhook_handler.LoggerHandler(ctx, body)
	return nil
}
