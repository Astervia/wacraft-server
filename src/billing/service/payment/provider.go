package payment

import (
	"time"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	"github.com/google/uuid"
)

// WebhookEvent represents a parsed payment webhook event.
type WebhookEvent struct {
	Type        string // e.g. "checkout.session.completed", "invoice.paid"
	ExternalID  string // Provider-specific ID (e.g. Stripe checkout session ID)
	PlanID      uuid.UUID
	UserID      uuid.UUID
	Scope       billing_model.Scope
	WorkspaceID *uuid.UUID
	PaymentMode billing_model.PaymentMode // "payment" or "subscription" — carried via metadata

	// Subscription mode fields
	SubscriptionID    string     // Provider-side subscription ID (for recurring plans)
	CustomerID        string     // Provider-side customer ID
	PeriodEnd         *time.Time // Current period end (for renewals via invoice.paid)
	CancelAtPeriodEnd bool       // Whether the subscription is set to cancel at period end
}

// SubscriptionDetails holds the current state of a subscription from the payment provider.
type SubscriptionDetails struct {
	Status            string     // "active", "canceled", "past_due", etc.
	CancelAtPeriodEnd bool
	CurrentPeriodEnd  time.Time
}

// CheckoutSessionStatus holds the current state of a checkout session from the payment provider.
type CheckoutSessionStatus struct {
	SessionStatus        string // "open", "complete", "expired"
	PaymentStatus        string // "paid", "unpaid", "no_payment_required"
	StripeSubscriptionID string
	CustomerID           string
}

// Provider defines the interface for payment processing integrations.
// Implement this interface to add support for new payment providers.
type Provider interface {
	// Name returns the provider identifier (e.g. "stripe", "paypal").
	Name() string

	// CreateCheckoutSession initiates a payment flow and returns a checkout URL.
	// paymentMode determines whether a one-time payment or recurring subscription is created.
	// For subscription mode, customerID may be provided to reuse an existing customer.
	CreateCheckoutSession(
		plan billing_entity.Plan,
		paymentMode billing_model.PaymentMode,
		userID uuid.UUID,
		userEmail string,
		customerID *string,
		scope billing_model.Scope,
		workspaceID *uuid.UUID,
		successURL string,
		cancelURL string,
	) (checkoutURL string, externalID string, err error)

	// CancelSubscription cancels an active payment-provider-side subscription.
	// For payment mode: no-op (nothing to cancel on provider side).
	// For subscription mode: sets cancel_at_period_end so the subscription
	// stays active until the current billing period ends.
	CancelSubscription(externalID string) error

	// ReactivateSubscription reverses a pending cancellation by setting
	// cancel_at_period_end back to false on the payment provider.
	ReactivateSubscription(externalID string) error

	// GetSubscriptionDetails fetches the current state of a subscription
	// from the payment provider for reconciliation purposes.
	GetSubscriptionDetails(subscriptionID string) (*SubscriptionDetails, error)

	// GetCheckoutSessionStatus fetches the current state of a checkout session
	// from the payment provider. Used to sync pending subscriptions.
	GetCheckoutSessionStatus(sessionID string) (*CheckoutSessionStatus, error)

	// VerifyWebhookSignature validates the authenticity of a webhook payload.
	VerifyWebhookSignature(payload []byte, signature string) error

	// ParseWebhookEvent extracts a structured event from a webhook payload.
	ParseWebhookEvent(payload []byte, signature string) (WebhookEvent, error)
}
