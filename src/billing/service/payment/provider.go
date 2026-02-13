package payment

import (
	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	"github.com/google/uuid"
)

// WebhookEvent represents a parsed payment webhook event.
type WebhookEvent struct {
	Type       string     // e.g. "checkout.session.completed", "payment_intent.succeeded"
	ExternalID string     // Provider-specific ID (e.g. Stripe checkout session ID)
	PlanID     uuid.UUID  // Plan being purchased
	UserID     uuid.UUID  // User who initiated the purchase
	Scope      billing_model.Scope
	WorkspaceID *uuid.UUID
}

// Provider defines the interface for payment processing integrations.
// Implement this interface to add support for new payment providers.
type Provider interface {
	// Name returns the provider identifier (e.g. "stripe", "paypal").
	Name() string

	// CreateCheckoutSession initiates a payment flow and returns a checkout URL.
	CreateCheckoutSession(
		plan billing_entity.Plan,
		userID uuid.UUID,
		scope billing_model.Scope,
		workspaceID *uuid.UUID,
		successURL string,
		cancelURL string,
	) (checkoutURL string, externalID string, err error)

	// CancelSubscription cancels an active payment-provider-side subscription.
	CancelSubscription(externalID string) error

	// VerifyWebhookSignature validates the authenticity of a webhook payload.
	VerifyWebhookSignature(payload []byte, signature string) error

	// ParseWebhookEvent extracts a structured event from a webhook payload.
	ParseWebhookEvent(payload []byte, signature string) (WebhookEvent, error)
}
