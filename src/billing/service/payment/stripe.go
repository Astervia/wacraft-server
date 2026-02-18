package payment

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/price"
	"github.com/stripe/stripe-go/v84/product"
	"github.com/stripe/stripe-go/v84/subscription"
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
	paymentMode billing_model.PaymentMode,
	userID uuid.UUID,
	userEmail string,
	customerID *string,
	scope billing_model.Scope,
	workspaceID *uuid.UUID,
	successURL string,
	cancelURL string,
) (string, string, error) {
	if env.StripeSecretKey == "" {
		return "", "", errors.New("stripe is not configured")
	}

	metadata := map[string]string{
		"plan_id":      plan.ID.String(),
		"user_id":      userID.String(),
		"scope":        string(scope),
		"payment_mode": string(paymentMode),
	}
	if workspaceID != nil {
		metadata["workspace_id"] = workspaceID.String()
	}

	if paymentMode == billing_model.PaymentModeSubscription {
		return s.createSubscriptionCheckout(plan, userEmail, customerID, metadata, successURL, cancelURL)
	}

	return s.createPaymentCheckout(plan, metadata, successURL, cancelURL)
}

// createPaymentCheckout creates a one-time payment checkout session (existing behavior).
func (s *StripeProvider) createPaymentCheckout(
	plan billing_entity.Plan,
	metadata map[string]string,
	successURL string,
	cancelURL string,
) (string, string, error) {
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

// createSubscriptionCheckout creates a recurring subscription checkout session.
func (s *StripeProvider) createSubscriptionCheckout(
	plan billing_entity.Plan,
	userEmail string,
	existingCustomerID *string,
	metadata map[string]string,
	successURL string,
	cancelURL string,
) (string, string, error) {
	// Ensure a Stripe Price exists for this plan.
	priceID, err := s.ensureStripePrice(plan)
	if err != nil {
		return "", "", fmt.Errorf("failed to ensure stripe price: %w", err)
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Metadata:   metadata,
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: metadata,
		},
	}

	// Set customer: reuse existing or let Stripe create via customer_email.
	// If the email is not a valid internet email (e.g. "su@sudo"), omit it
	// and let Stripe's checkout page collect the email from the user.
	if existingCustomerID != nil && *existingCustomerID != "" {
		params.Customer = existingCustomerID
	} else if isValidInternetEmail(userEmail) {
		params.CustomerEmail = stripe.String(userEmail)
	}

	sess, err := session.New(params)
	if err != nil {
		return "", "", fmt.Errorf("failed to create stripe subscription checkout: %w", err)
	}

	return sess.URL, sess.ID, nil
}

// ensureStripePrice lazily creates a Stripe Product + Price for subscription plans
// and caches the IDs on the plan entity.
func (s *StripeProvider) ensureStripePrice(plan billing_entity.Plan) (string, error) {
	if plan.StripePriceID != nil && *plan.StripePriceID != "" {
		return *plan.StripePriceID, nil
	}

	// Create a Stripe Product.
	prodParams := &stripe.ProductParams{
		Name: stripe.String(plan.Name),
		Metadata: map[string]string{
			"plan_id": plan.ID.String(),
		},
	}
	if plan.Description != nil {
		prodParams.Description = plan.Description
	}

	prod, err := product.New(prodParams)
	if err != nil {
		return "", fmt.Errorf("failed to create stripe product: %w", err)
	}

	// Create a Stripe Price with recurring interval based on duration_days.
	priceParams := &stripe.PriceParams{
		Product:    stripe.String(prod.ID),
		UnitAmount: stripe.Int64(plan.PriceCents),
		Currency:   stripe.String(plan.Currency),
		Recurring: &stripe.PriceRecurringParams{
			Interval:      stripe.String(string(stripe.PriceRecurringIntervalDay)),
			IntervalCount: stripe.Int64(int64(plan.DurationDays)),
		},
	}

	p, err := price.New(priceParams)
	if err != nil {
		return "", fmt.Errorf("failed to create stripe price: %w", err)
	}

	// Persist the IDs back to the plan in the database.
	database.DB.Model(&billing_entity.Plan{}).
		Where("id = ?", plan.ID).
		Updates(map[string]any{
			"stripe_price_id":   p.ID,
			"stripe_product_id": prod.ID,
		})

	return p.ID, nil
}

func (s *StripeProvider) CancelSubscription(externalID string) error {
	if externalID == "" {
		return nil // No-op for payment mode
	}

	// Set cancel_at_period_end so the subscription stays active until the period ends.
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	_, err := subscription.Update(externalID, params)
	if err != nil {
		return fmt.Errorf("failed to cancel stripe subscription: %w", err)
	}
	return nil
}

func (s *StripeProvider) ReactivateSubscription(externalID string) error {
	if externalID == "" {
		return nil
	}

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}
	_, err := subscription.Update(externalID, params)
	if err != nil {
		return fmt.Errorf("failed to reactivate stripe subscription: %w", err)
	}
	return nil
}

func (s *StripeProvider) GetSubscriptionDetails(subscriptionID string) (*SubscriptionDetails, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe subscription: %w", err)
	}

	// In Stripe v84, CurrentPeriodEnd is on subscription items, not the subscription itself.
	var periodEnd time.Time
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		periodEnd = time.Unix(sub.Items.Data[0].CurrentPeriodEnd, 0)
	}

	return &SubscriptionDetails{
		Status:            string(sub.Status),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
		CurrentPeriodEnd:  periodEnd,
	}, nil
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

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := sess.UnmarshalJSON(event.Data.Raw); err != nil {
			return result, fmt.Errorf("failed to parse checkout session: %w", err)
		}

		result.ExternalID = sess.ID
		s.extractMetadata(&result, sess.Metadata)

		if sess.Customer != nil {
			result.CustomerID = sess.Customer.ID
		}
		if sess.Subscription != nil {
			result.SubscriptionID = sess.Subscription.ID
		}

	case "invoice.paid":
		var inv stripe.Invoice
		if err := inv.UnmarshalJSON(event.Data.Raw); err != nil {
			return result, fmt.Errorf("failed to parse invoice: %w", err)
		}

		// In Stripe v84, subscription info is under Parent.SubscriptionDetails
		if inv.Parent != nil && inv.Parent.SubscriptionDetails != nil {
			if inv.Parent.SubscriptionDetails.Subscription != nil {
				result.SubscriptionID = inv.Parent.SubscriptionDetails.Subscription.ID
			}
			if inv.Parent.SubscriptionDetails.Metadata != nil {
				s.extractMetadata(&result, inv.Parent.SubscriptionDetails.Metadata)
			}
		}
		if inv.Customer != nil {
			result.CustomerID = inv.Customer.ID
		}
		// PeriodEnd from the invoice lines gives the current billing period end.
		if len(inv.Lines.Data) > 0 {
			periodEnd := time.Unix(inv.Lines.Data[0].Period.End, 0)
			result.PeriodEnd = &periodEnd
		}

	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := sub.UnmarshalJSON(event.Data.Raw); err != nil {
			return result, fmt.Errorf("failed to parse subscription: %w", err)
		}

		result.SubscriptionID = sub.ID
		result.ExternalID = sub.ID
		s.extractMetadata(&result, sub.Metadata)
		if sub.Customer != nil {
			result.CustomerID = sub.Customer.ID
		}
		result.CancelAtPeriodEnd = sub.CancelAtPeriodEnd

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := sub.UnmarshalJSON(event.Data.Raw); err != nil {
			return result, fmt.Errorf("failed to parse subscription: %w", err)
		}

		result.SubscriptionID = sub.ID
		result.ExternalID = sub.ID
		s.extractMetadata(&result, sub.Metadata)
		if sub.Customer != nil {
			result.CustomerID = sub.Customer.ID
		}
	}

	return result, nil
}

// extractMetadata extracts plan/user/scope metadata from a Stripe metadata map.
func (s *StripeProvider) extractMetadata(result *WebhookEvent, metadata map[string]string) {
	if planIDStr, ok := metadata["plan_id"]; ok {
		if id, err := uuid.Parse(planIDStr); err == nil {
			result.PlanID = id
		}
	}
	if userIDStr, ok := metadata["user_id"]; ok {
		if id, err := uuid.Parse(userIDStr); err == nil {
			result.UserID = id
		}
	}
	if scopeStr, ok := metadata["scope"]; ok {
		result.Scope = billing_model.Scope(scopeStr)
	}
	if wsIDStr, ok := metadata["workspace_id"]; ok {
		if id, err := uuid.Parse(wsIDStr); err == nil {
			result.WorkspaceID = &id
		}
	}
	if modeStr, ok := metadata["payment_mode"]; ok {
		result.PaymentMode = billing_model.PaymentMode(modeStr)
	}
}

// isValidInternetEmail checks that the email is RFC 5322 valid AND has a
// domain with at least one dot (i.e. a proper internet domain, not a local
// hostname like "su@sudo"). Stripe rejects emails without a TLD.
func isValidInternetEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	parts := strings.SplitN(addr.Address, "@", 2)
	return len(parts) == 2 && strings.Contains(parts[1], ".")
}

// ActiveProvider is the configured payment provider. Set during server initialization.
var ActiveProvider Provider
