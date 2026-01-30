package webhook_config

import (
	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	phone_config_service "github.com/Astervia/wacraft-server/src/phone-config/service"
	webhook_handler "github.com/Astervia/wacraft-server/src/webhook-in/handler"
	wh_model "github.com/Rfluid/whatsapp-cloud-api/src/webhook"
	auth_middleware "github.com/Rfluid/whatsapp-webhook-server/src/auth/middleware"
	server_service "github.com/Rfluid/whatsapp-webhook-server/src/server/service"
	webhook_model "github.com/Rfluid/whatsapp-webhook-server/src/webhook/model"
	webhook_service "github.com/Rfluid/whatsapp-webhook-server/src/webhook/service"
	"github.com/gofiber/fiber/v2"
	"github.com/pterm/pterm"
)

// LegacyHook is the original webhook config using environment variables.
// Used for backwards compatibility during migration.
var LegacyHook = webhook_service.Config{
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

// ServeWebhook registers webhook endpoints for all phone configs and the legacy endpoint.
func ServeWebhook(app *fiber.App) {
	// Register legacy webhook at /webhook-in for backwards compatibility
	if env.WabaAccessToken != "" {
		pterm.DefaultLogger.Info("Registering legacy webhook at /webhook-in")
		server := server_service.NewConfig(app, "/webhook-in")
		server_service.Bootstrap(server, &LegacyHook)
	}

	// Register per-phone-config webhooks at /webhook-in/:phone_number_id
	registerPhoneConfigWebhooks(app)
}

// registerPhoneConfigWebhooks registers webhook endpoints for each active phone config.
func registerPhoneConfigWebhooks(app *fiber.App) {
	var phoneConfigs []phone_config_entity.PhoneConfig
	if err := database.DB.Where("is_active = true").Find(&phoneConfigs).Error; err != nil {
		pterm.DefaultLogger.Warn("Failed to load phone configs for webhook registration: " + err.Error())
		return
	}

	for _, pc := range phoneConfigs {
		registerPhoneConfigWebhook(app, pc)
	}

	pterm.DefaultLogger.Info("Registered webhooks for phone configs")
}

// registerPhoneConfigWebhook registers a webhook endpoint for a specific phone config.
func registerPhoneConfigWebhook(app *fiber.App, pc phone_config_entity.PhoneConfig) {
	path := "/" + pc.WabaID

	appSecret := phone_config_service.GetMetaAppSecret(&pc)
	verifyToken := phone_config_service.GetVerifyToken(&pc)

	hook := webhook_service.Config{
		Path: path,
		ChangeHandlers: []webhook_model.ChangeHandler{
			webhook_handler.CreateMessageHandlerForPhoneConfig(pc.ID, pc.WabaID),
		},
		CtxHandler: defaultCtxHandler,
		PostMiddlewares: []func(ctx *fiber.Ctx) error{
			func(ctx *fiber.Ctx) error {
				if appSecret == "" {
					return ctx.Next()
				}
				return auth_middleware.VerifyMetaSignature(appSecret)(ctx)
			},
		},
		GetMiddlewares: []func(ctx *fiber.Ctx) error{
			func(ctx *fiber.Ctx) error {
				if verifyToken == "" {
					return ctx.Next()
				}
				return auth_middleware.MetaVerificationRequestToken(verifyToken)(ctx)
			},
		},
	}

	server := server_service.NewConfig(app, "/webhook-in")
	server_service.Bootstrap(server, &hook)

	pterm.DefaultLogger.Info("Registered webhook for phone config: " + pc.Name + " at /webhook-in/" + pc.WabaID)
}

func defaultCtxHandler(ctx *fiber.Ctx, body *wh_model.WebhookBody) error {
	return nil
}
