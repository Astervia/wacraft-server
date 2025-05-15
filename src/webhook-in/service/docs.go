package webhook_service

import (
	_ "github.com/Rfluid/whatsapp-cloud-api/src/webhook/model"
	"github.com/gofiber/fiber/v2"
)

//	@Summary		Handles Webhooks.
//	@Description	Executes the context handler then the change handlers. If any error is thrown this function will also throw an error.
//	@Tags			Webhook In
//	@Accept			json
//	@Produce		json
//	@Param			input	body	webhook_model.WebhookBody	true	"Content sent by WhatsApp Cloud API."
//	@Success		200		"Valid webhook endpoint."
//	@Router			/webhook-in [post]
func postWebHookDocs(ctx *fiber.Ctx) error {
	return nil
}

//	@Summary		Verify Webhook.
//	@Description	Used by meta to verify if it is a valid webhook endpoint.
//	@Tags			Webhook In
//	@Accept			json
//	@Produce		json
//	@Param			hub.mode			query		string	true	"Subscription mode, always set to 'subscribe'"
//	@Param			hub.challenge		query		int		true	"A challenge integer that must be returned to confirm the webhook"
//	@Param			hub.verify_token	query		string	true	"A string used for validation, defined in the Webhooks setup in the App Dashboard"
//	@Success		200					{string}	string	"hub.challenge returned as a string."
//	@Router			/webhook-in [get]
func getWebHookDocs(ctx *fiber.Ctx) error {
	return nil
}

//	@Security	ApiKeyAuth
