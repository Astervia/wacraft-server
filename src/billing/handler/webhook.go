package billing_handler

import (
	"fmt"

	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
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
		// Default to payment mode if not specified in metadata
		paymentMode := event.PaymentMode
		if paymentMode == "" {
			paymentMode = billing_model.PaymentModePayment
		}

		_, err := billing_service.ActivateSubscription(
			event.PlanID,
			event.Scope,
			event.UserID,
			event.WorkspaceID,
			payment.ActiveProvider.Name(),
			event.ExternalID,
			paymentMode,
			event.SubscriptionID,
			event.CustomerID,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("failed to activate subscription", err, "billing").Send(),
			)
		}

	case "invoice.paid":
		// Subscription renewal: extend the subscription's ExpiresAt.
		// The initial invoice (same period as checkout) is handled gracefully —
		// if the subscription isn't found yet, we log and move on.
		if event.SubscriptionID != "" && event.PeriodEnd != nil {
			if err := billing_service.RenewSubscription(event.SubscriptionID, *event.PeriodEnd); err != nil {
				pterm.DefaultLogger.Warn("invoice.paid renewal skipped: " + err.Error())
			}
		}

	case "customer.subscription.deleted":
		// Stripe has fully ended the subscription (period ended after cancellation).
		if event.SubscriptionID != "" {
			if err := billing_service.MarkSubscriptionCancelled(event.SubscriptionID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(
					common_model.NewApiError("failed to mark subscription cancelled", err, "billing").Send(),
				)
			}
		}

	case "customer.subscription.updated":
		pterm.DefaultLogger.Info(fmt.Sprintf(
			"Subscription %s updated: cancel_at_period_end=%v",
			event.SubscriptionID, event.CancelAtPeriodEnd,
		))
		if event.SubscriptionID != "" {
			if err := billing_service.SyncCancelAtPeriodEnd(event.SubscriptionID, event.CancelAtPeriodEnd); err != nil {
				pterm.DefaultLogger.Warn("subscription.updated sync failed: " + err.Error())
			}
		}
	}

	return c.SendStatus(fiber.StatusOK)
}
