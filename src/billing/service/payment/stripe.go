package payment

import (
	"errors"
	"fmt"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/webhook"
)

// StripeProvider implements the Provider interface for Stripe payments.
type StripeProvider struct{}

func NewStripeProvider() *StripeProvider {
	stripe.Key = env.StripeSecretKey
	return &StripeProvider{}
}

func (s *StripeProvider) Name() string {
	return "stripe"
}

func (s *StripeProvider) CreateCheckoutSession(
	plan billing_entity.Plan,
	userID uuid.UUID,
	scope billing_model.Scope,
	workspaceID *uuid.UUID,
	successURL string,
	cancelURL string,
) (string, string, error) {
	if env.StripeSecretKey == "" {
		return "", "", errors.New("stripe is not configured")
	}

	metadata := map[string]string{
		"plan_id": plan.ID.String(),
		"user_id": userID.String(),
		"scope":   string(scope),
	}
	if workspaceID != nil {
		metadata["workspace_id"] = workspaceID.String()
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(plan.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(plan.Name),
						Description: plan.Description,
					},
					UnitAmount: stripe.Int64(plan.PriceCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Metadata:   metadata,
	}

	sess, err := session.New(params)
	if err != nil {
		return "", "", fmt.Errorf("failed to create stripe checkout session: %w", err)
	}

	return sess.URL, sess.ID, nil
}

func (s *StripeProvider) CancelSubscription(externalID string) error {
	// For one-time payments, there's no recurring subscription to cancel on Stripe's side.
	// This is a no-op for payment mode. If recurring billing is added later,
	// this would call stripe subscription cancellation.
	return nil
}

func (s *StripeProvider) VerifyWebhookSignature(payload []byte, signature string) error {
	if env.StripeWebhookSecret == "" {
		return errors.New("stripe webhook secret is not configured")
	}
	_, err := webhook.ConstructEvent(payload, signature, env.StripeWebhookSecret)
	return err
}

func (s *StripeProvider) ParseWebhookEvent(payload []byte, signature string) (WebhookEvent, error) {
	if env.StripeWebhookSecret == "" {
		return WebhookEvent{}, errors.New("stripe webhook secret is not configured")
	}

	event, err := webhook.ConstructEvent(payload, signature, env.StripeWebhookSecret)
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	result := WebhookEvent{
		Type: string(event.Type),
	}

	// For checkout.session.completed, extract metadata
	if event.Type == "checkout.session.completed" {
		var sess stripe.CheckoutSession
		if err := sess.UnmarshalJSON(event.Data.Raw); err != nil {
			return result, fmt.Errorf("failed to parse checkout session: %w", err)
		}

		result.ExternalID = sess.ID

		if planIDStr, ok := sess.Metadata["plan_id"]; ok {
			if id, err := uuid.Parse(planIDStr); err == nil {
				result.PlanID = id
			}
		}
		if userIDStr, ok := sess.Metadata["user_id"]; ok {
			if id, err := uuid.Parse(userIDStr); err == nil {
				result.UserID = id
			}
		}
		if scopeStr, ok := sess.Metadata["scope"]; ok {
			result.Scope = billing_model.Scope(scopeStr)
		}
		if wsIDStr, ok := sess.Metadata["workspace_id"]; ok {
			if id, err := uuid.Parse(wsIDStr); err == nil {
				result.WorkspaceID = &id
			}
		}
	}

	return result, nil
}

// ActiveProvider is the configured payment provider. Set during server initialization.
var ActiveProvider Provider
