package billing_service

import (
	"errors"
	"fmt"
	"time"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	"github.com/Astervia/wacraft-server/src/billing/service/payment"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/google/uuid"
)

// CreateManualSubscription creates a subscription for admin-managed (manual) plans.
func CreateManualSubscription(data billing_model.CreateManualSubscription) (billing_entity.Subscription, error) {
	// Validate scope + workspace
	if data.Scope == billing_model.ScopeWorkspace && data.WorkspaceID == nil {
		return billing_entity.Subscription{}, errors.New("workspace_id is required for workspace-scoped subscriptions")
	}

	// Fetch the plan
	var plan billing_entity.Plan
	if err := database.DB.First(&plan, data.PlanID).Error; err != nil {
		return billing_entity.Subscription{}, errors.New("plan not found")
	}

	now := time.Now()
	sub := billing_entity.Subscription{
		PlanID:             data.PlanID,
		Scope:              data.Scope,
		UserID:             data.UserID,
		WorkspaceID:        data.WorkspaceID,
		ThroughputOverride: data.ThroughputOverride,
		StartsAt:           now,
		ExpiresAt:          now.AddDate(0, 0, plan.DurationDays),
		PaymentProvider:    "manual",
	}

	if err := database.DB.Create(&sub).Error; err != nil {
		return sub, err
	}

	// Invalidate cache for affected scope
	invalidateForSubscription(&sub)

	return sub, nil
}

// ActivateSubscription creates a subscription after successful payment.
func ActivateSubscription(
	planID uuid.UUID,
	scope billing_model.Scope,
	userID uuid.UUID,
	workspaceID *uuid.UUID,
	provider string,
	externalID string,
	paymentMode billing_model.PaymentMode,
	stripeSubscriptionID string,
	customerID string,
) (billing_entity.Subscription, error) {
	var plan billing_entity.Plan
	if err := database.DB.First(&plan, planID).Error; err != nil {
		return billing_entity.Subscription{}, errors.New("plan not found")
	}

	now := time.Now()
	sub := billing_entity.Subscription{
		PlanID:            planID,
		Scope:             scope,
		UserID:            userID,
		WorkspaceID:       workspaceID,
		StartsAt:          now,
		ExpiresAt:         now.AddDate(0, 0, plan.DurationDays),
		PaymentProvider:   provider,
		PaymentExternalID: &externalID,
		PaymentMode:       paymentMode,
	}

	// For subscription mode, store the Stripe subscription ID
	if stripeSubscriptionID != "" {
		sub.StripeSubscriptionID = &stripeSubscriptionID
	}

	if err := database.DB.Create(&sub).Error; err != nil {
		return sub, err
	}

	// Persist the Stripe Customer ID on the user (if not already set)
	if customerID != "" {
		database.DB.Model(&user_entity.User{}).
			Where("id = ? AND (stripe_customer_id IS NULL OR stripe_customer_id = '')", userID).
			Update("stripe_customer_id", customerID)
	}

	invalidateForSubscription(&sub)

	return sub, nil
}

// CancelSubscription cancels a subscription.
// For payment mode: immediately sets CancelledAt (subscription becomes inactive).
// For subscription mode: calls the payment provider to cancel at period end;
// the subscription stays active until ExpiresAt and CancelledAt is set later
// by the customer.subscription.deleted webhook.
// CancelSubscription cancels a subscription.
// Only renewable (subscription mode) subscriptions can be cancelled.
// Cancellation calls the payment provider to stop renewal; the subscription
// stays active until ExpiresAt. CancelledAt is set later by the
// customer.subscription.deleted webhook.
// One-time (payment mode) subscriptions cannot be cancelled — they simply
// expire at ExpiresAt.
func CancelSubscription(subscriptionID uuid.UUID, userID uuid.UUID) error {
	var sub billing_entity.Subscription
	if err := database.DB.First(&sub, subscriptionID).Error; err != nil {
		return errors.New("subscription not found")
	}

	// Only the owner can cancel
	if sub.UserID != userID {
		return errors.New("unauthorized: you can only cancel your own subscriptions")
	}

	if sub.CancelledAt != nil {
		return errors.New("subscription is already cancelled")
	}

	// Only subscription (renewable) mode can be cancelled
	if sub.PaymentMode != billing_model.PaymentModeSubscription {
		return errors.New("one-time payment subscriptions cannot be cancelled — they expire naturally")
	}

	// Cancel at period end via payment provider
	if sub.StripeSubscriptionID != nil && payment.ActiveProvider != nil {
		if err := payment.ActiveProvider.CancelSubscription(*sub.StripeSubscriptionID); err != nil {
			return fmt.Errorf("failed to cancel on payment provider: %w", err)
		}
	}

	// Mark as pending cancellation so the API can expose this state.
	sub.CancelAtPeriodEnd = true
	if err := database.DB.Model(&sub).Update("cancel_at_period_end", true).Error; err != nil {
		return fmt.Errorf("failed to update cancel_at_period_end: %w", err)
	}

	// The subscription remains active until ExpiresAt.
	// CancelledAt will be set when Stripe fires customer.subscription.deleted.
	return nil
}

// ReactivateSubscription reverses a pending cancellation, re-enabling auto-renewal.
// Only works for subscription-mode subscriptions with cancel_at_period_end=true.
func ReactivateSubscription(subscriptionID uuid.UUID, userID uuid.UUID) error {
	var sub billing_entity.Subscription
	if err := database.DB.First(&sub, subscriptionID).Error; err != nil {
		return errors.New("subscription not found")
	}

	if sub.UserID != userID {
		return errors.New("unauthorized: you can only reactivate your own subscriptions")
	}

	if sub.CancelledAt != nil {
		return errors.New("subscription is already fully cancelled and cannot be reactivated")
	}

	if !sub.CancelAtPeriodEnd {
		return errors.New("subscription is not pending cancellation")
	}

	if sub.PaymentMode != billing_model.PaymentModeSubscription {
		return errors.New("only recurring subscriptions can be reactivated")
	}

	// Reactivate on the payment provider
	if sub.StripeSubscriptionID != nil && payment.ActiveProvider != nil {
		if err := payment.ActiveProvider.ReactivateSubscription(*sub.StripeSubscriptionID); err != nil {
			return fmt.Errorf("failed to reactivate on payment provider: %w", err)
		}
	}

	// Clear the pending cancellation flag
	if err := database.DB.Model(&sub).Update("cancel_at_period_end", false).Error; err != nil {
		return fmt.Errorf("failed to update cancel_at_period_end: %w", err)
	}

	return nil
}

// RenewSubscription extends an existing subscription's ExpiresAt for the next billing period.
// Called when Stripe fires invoice.paid for a recurring subscription.
func RenewSubscription(stripeSubscriptionID string, periodEnd time.Time) error {
	var sub billing_entity.Subscription
	if err := database.DB.
		Where("stripe_subscription_id = ? AND cancelled_at IS NULL", stripeSubscriptionID).
		First(&sub).Error; err != nil {
		return errors.New("active subscription not found for stripe subscription ID")
	}

	sub.ExpiresAt = periodEnd
	if err := database.DB.Save(&sub).Error; err != nil {
		return err
	}

	invalidateForSubscription(&sub)
	return nil
}

// MarkSubscriptionCancelled sets CancelledAt on a subscription when the payment provider
// confirms the subscription has ended (e.g. Stripe customer.subscription.deleted).
func MarkSubscriptionCancelled(stripeSubscriptionID string) error {
	var sub billing_entity.Subscription
	if err := database.DB.
		Where("stripe_subscription_id = ?", stripeSubscriptionID).
		First(&sub).Error; err != nil {
		return errors.New("subscription not found for stripe subscription ID")
	}

	if sub.CancelledAt != nil {
		return nil // Already cancelled, idempotent
	}

	now := time.Now()
	sub.CancelledAt = &now
	if err := database.DB.Save(&sub).Error; err != nil {
		return err
	}

	invalidateForSubscription(&sub)
	return nil
}

// SyncCancelAtPeriodEnd updates the cancel_at_period_end flag on a subscription
// to match the value reported by the payment provider (e.g. Stripe customer.subscription.updated).
func SyncCancelAtPeriodEnd(stripeSubscriptionID string, cancelAtPeriodEnd bool) error {
	result := database.DB.
		Model(&billing_entity.Subscription{}).
		Where("stripe_subscription_id = ?", stripeSubscriptionID).
		Update("cancel_at_period_end", cancelAtPeriodEnd)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("subscription not found for stripe subscription ID")
	}
	return nil
}

// GetActiveSubscriptions returns all active subscriptions for a scope.
func GetActiveSubscriptions(scope billing_model.Scope, userID *uuid.UUID, workspaceID *uuid.UUID) ([]billing_entity.Subscription, error) {
	now := time.Now()
	query := database.DB.
		Preload("Plan").
		Where("scope = ? AND starts_at <= ? AND expires_at > ? AND cancelled_at IS NULL", scope, now, now)

	if scope == billing_model.ScopeUser && userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else if scope == billing_model.ScopeWorkspace && workspaceID != nil {
		query = query.Where("workspace_id = ?", *workspaceID)
	}

	var subs []billing_entity.Subscription
	if err := query.Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}

func invalidateForSubscription(sub *billing_entity.Subscription) {
	InvalidateCache(sub.Scope, sub.UserID)
	if sub.WorkspaceID != nil {
		InvalidateCache(billing_model.ScopeWorkspace, *sub.WorkspaceID)
	}
}
