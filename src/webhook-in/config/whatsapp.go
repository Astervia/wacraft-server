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

	// Register webhook for all phone configs at /webhook-in/:waba_id
	registerWabaWebhook(app)
}

// registerWabaWebhook registers a single webhook endpoint at /webhook-in/:waba_id.
func registerWabaWebhook(app *fiber.App) {
	hook := webhook_service.Config{
		Path: "/:waba_id",
		ChangeHandlers: []webhook_model.ChangeHandler{
			webhook_handler.PhoneConfigMessageHandler,
		},
		CtxHandler: defaultCtxHandler,
		PostMiddlewares: []func(ctx *fiber.Ctx) error{
			func(ctx *fiber.Ctx) error {
				phoneConfig, err := requirePhoneConfig(ctx)
				if err != nil {
					return err
				}
				appSecret := phone_config_service.GetMetaAppSecret(phoneConfig)
				if appSecret == "" {
					return ctx.Next()
				}
				return auth_middleware.VerifyMetaSignature(appSecret)(ctx)
			},
		},
		GetMiddlewares: []func(ctx *fiber.Ctx) error{
			func(ctx *fiber.Ctx) error {
				phoneConfig, err := requirePhoneConfig(ctx)
				if err != nil {
					return err
				}
				verifyToken := phone_config_service.GetVerifyToken(phoneConfig)
				if verifyToken == "" {
					return ctx.Next()
				}
				return auth_middleware.MetaVerificationRequestToken(verifyToken)(ctx)
			},
		},
	}

	server := server_service.NewConfig(app, "/webhook-in")
	server_service.Bootstrap(server, &hook)

	pterm.DefaultLogger.Info("Registered webhook for all phone configs at /webhook-in/:waba_id")
}

func defaultCtxHandler(ctx *fiber.Ctx, body *wh_model.WebhookBody) error {
	return nil
}

func requirePhoneConfig(ctx *fiber.Ctx) (*phone_config_entity.PhoneConfig, error) {
	if phoneConfig, ok := ctx.Locals(webhook_handler.PhoneConfigCtxKey).(*phone_config_entity.PhoneConfig); ok && phoneConfig != nil {
		return phoneConfig, nil
	}

	wabaID := ctx.Params("waba_id")
	if wabaID == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "missing waba_id")
	}

	var phoneConfig phone_config_entity.PhoneConfig
	if err := database.DB.Where("waba_id = ? AND is_active = true", wabaID).First(&phoneConfig).Error; err != nil {
		pterm.DefaultLogger.Warn("Phone config not found for webhook waba_id " + wabaID + ": " + err.Error())
		return nil, fiber.NewError(fiber.StatusNotFound, "phone config not found")
	}

	ctx.Locals(webhook_handler.PhoneConfigCtxKey, &phoneConfig)
	return &phoneConfig, nil
}
