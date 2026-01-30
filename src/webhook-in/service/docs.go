package webhook_service

import (
	_ "github.com/Rfluid/whatsapp-cloud-api/src/webhook"
	"github.com/gofiber/fiber/v2"
)

// postWebHookDocs handles incoming webhook events from WhatsApp Cloud API.
//
//	@Summary		Handle webhook events
//	@Description	Processes the context handler and all change handlers. If an error occurs, it is returned.
//	@Tags			Webhook In
//	@Accept			json
//	@Produce		json
//	@Param			waba_id	path		string								true	"Phone number ID provided by Meta"
//	@Param			input	body		webhook.WebhookBody	true	"Content sent by WhatsApp Cloud API"
//	@Success		200		{string}	string						"Valid webhook received"
//	@Router			/webhook-in/{waba_id} [post]
func postWebHookDocs(_ *fiber.Ctx) error {
	return nil
}

// getWebHookDocs verifies webhook configuration for WhatsApp Cloud API.
//
//	@Summary		Verify webhook endpoint
//	@Description	Used by Meta to verify the validity of the webhook endpoint.
//	@Tags			Webhook In
//	@Accept			json
//	@Produce		json
//	@Param			waba_id	path		string								true	"Phone number ID provided by Meta"
//	@Param			hub.mode			query		string	true	"Subscription mode (should be 'subscribe')"
//	@Param			hub.challenge		query		int		true	"Challenge token returned by the endpoint"
//	@Param			hub.verify_token	query		string	true	"Verification token defined in Meta dashboard"
//	@Success		200					{string}	string	"hub.challenge echoed back"
//	@Router			/webhook-in/{waba_id} [get]
func getWebHookDocs(_ *fiber.Ctx) error {
	return nil
}
