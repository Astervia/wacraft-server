package billing_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	billing_service "github.com/Astervia/wacraft-server/src/billing/service"
	"github.com/Astervia/wacraft-server/src/billing/service/payment"
	"github.com/gofiber/fiber/v2"
	"github.com/pterm/pterm"
)

// StripeWebhook handles incoming Stripe webhook events.
//
//	@Summary		Handle Stripe webhook
//	@Description	Receives and processes Stripe webhook events. Validates the payload using the Stripe-Signature header. No authentication required.
//	@Tags			Billing Webhook
//	@Accept			json
//	@Produce		json
//	@Param			Stripe-Signature	header		string							true	"Stripe webhook signature"
//	@Success		200					{string}	string							"OK"
//	@Failure		400					{object}	common_model.DescriptiveError	"Invalid webhook payload"
//	@Failure		500					{object}	common_model.DescriptiveError	"Internal server error"
//	@Failure		503					{object}	common_model.DescriptiveError	"Payment provider not configured"
//	@Router			/billing/webhook/stripe [post]
func StripeWebhook(c *fiber.Ctx) error {
	if payment.ActiveProvider == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(
			common_model.NewApiError("payment provider is not configured", nil, "billing").Send(),
		)
	}

	payload := c.Body()
	signature := c.Get("Stripe-Signature")

	event, err := payment.ActiveProvider.ParseWebhookEvent(payload, signature)
	if err != nil {
		pterm.DefaultLogger.Error("Stripe webhook failed: " + err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("invalid webhook payload", err, "billing").Send(),
		)
	}

	switch event.Type {
	case "checkout.session.completed":
		_, err := billing_service.ActivateSubscription(
			event.PlanID,
			event.Scope,
			event.UserID,
			event.WorkspaceID,
			payment.ActiveProvider.Name(),
			event.ExternalID,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("failed to activate subscription", err, "billing").Send(),
			)
		}
	}

	return c.SendStatus(fiber.StatusOK)
}
