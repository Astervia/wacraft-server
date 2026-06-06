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

// CreatePendingSubscription creates a subscription in "pending" state at checkout time.
// This ensures a local record exists before the webhook fires, enabling sync recovery
// if the webhook fails.
func CreatePendingSubscription(
	planID uuid.UUID,
	scope billing_model.Scope,
	userID uuid.UUID,
	workspaceID *uuid.UUID,
	provider string,
	externalID string,
	paymentMode billing_model.PaymentMode,
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
		Status:            billing_model.SubscriptionStatusPending,
	}

	if err := database.DB.Create(&sub).Error; err != nil {
		return sub, err
	}

	// Do NOT invalidate throughput cache — pending subs don't affect it.
	return sub, nil
}

// ActivateSubscription activates a subscription after successful payment.
// It first looks for a pending subscription by payment_external_id; if found,
// it transitions it to active. If not found, it falls back to creating a new
// active subscription (backward compatibility for pre-migration webhooks).
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
	// Idempotency: if already active for this externalID, return it
	var existing billing_entity.Subscription
	if err := database.DB.
		Where("payment_external_id = ? AND status = ?", externalID, billing_model.SubscriptionStatusActive).
		First(&existing).Error; err == nil {
		return existing, nil
	}

	// Look for a pending subscription to activate
	var pending billing_entity.Subscription
	if err := database.DB.
		Where("payment_external_id = ? AND status = ?", externalID, billing_model.SubscriptionStatusPending).
		First(&pending).Error; err == nil {
		// Found pending — activate it
		var plan billing_entity.Plan
		if err := database.DB.First(&plan, pending.PlanID).Error; err != nil {
			return billing_entity.Subscription{}, errors.New("plan not found")
		}

		now := time.Now()
		pending.Status = billing_model.SubscriptionStatusActive
		pending.StartsAt = now
		pending.ExpiresAt = now.AddDate(0, 0, plan.DurationDays)
		if stripeSubscriptionID != "" {
			pending.StripeSubscriptionID = &stripeSubscriptionID
		}

		if err := database.DB.Save(&pending).Error; err != nil {
			return pending, err
		}

		// Persist the Stripe Customer ID on the user (if not already set)
		if customerID != "" {
			database.DB.Model(&user_entity.User{}).
				Where("id = ? AND (stripe_customer_id IS NULL OR stripe_customer_id = '')", userID).
				Update("stripe_customer_id", customerID)
		}

		invalidateForSubscription(&pending)
		return pending, nil
	}

	// Fallback: no pending record found — create a new active subscription
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
		Status:            billing_model.SubscriptionStatusActive,
	}

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
func CancelSubscription(subscriptionID uuid.UUID, userID uuid.UUID, workspaceID *uuid.UUID) error {
	var sub billing_entity.Subscription
	if err := database.DB.First(&sub, subscriptionID).Error; err != nil {
		return errors.New("subscription not found")
	}

	// Ownership check: workspace-scoped or user-scoped
	if workspaceID != nil {
		if sub.WorkspaceID == nil || *sub.WorkspaceID != *workspaceID {
			return errors.New("unauthorized: subscription does not belong to this workspace")
		}
	} else if sub.UserID != userID {
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
func ReactivateSubscription(subscriptionID uuid.UUID, userID uuid.UUID, workspaceID *uuid.UUID) error {
	var sub billing_entity.Subscription
	if err := database.DB.First(&sub, subscriptionID).Error; err != nil {
		return errors.New("subscription not found")
	}

	// Ownership check: workspace-scoped or user-scoped
	if workspaceID != nil {
		if sub.WorkspaceID == nil || *sub.WorkspaceID != *workspaceID {
			return errors.New("unauthorized: subscription does not belong to this workspace")
		}
	} else if sub.UserID != userID {
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
	sub.Status = billing_model.SubscriptionStatusCancelled
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

// SyncSubscription fetches the current subscription state from the payment provider
// and reconciles the local DB record.
//
// Three sync paths:
//   - Pending subscription (any mode): uses GetCheckoutSessionStatus via PaymentExternalID.
//     If paid, activates it. If session expired/unpaid, marks cancelled.
//   - Active subscription-mode with StripeSubscriptionID: existing behavior (fetch from Stripe subscription API).
//   - Active subscription-mode without StripeSubscriptionID: error (shouldn't happen).
func SyncSubscription(subscriptionID uuid.UUID, userID uuid.UUID, workspaceID *uuid.UUID) (billing_entity.Subscription, error) {
	var sub billing_entity.Subscription
	if err := database.DB.Preload("Plan").First(&sub, subscriptionID).Error; err != nil {
		return billing_entity.Subscription{}, errors.New("subscription not found")
	}

	// Ownership check: workspace-scoped or user-scoped
	if workspaceID != nil {
		if sub.WorkspaceID == nil || *sub.WorkspaceID != *workspaceID {
			return billing_entity.Subscription{}, errors.New("unauthorized: subscription does not belong to this workspace")
		}
	} else if sub.UserID != userID {
		return billing_entity.Subscription{}, errors.New("unauthorized: you can only sync your own subscriptions")
	}

	// Path 1: Pending subscription — sync via checkout session status
	if sub.Status == billing_model.SubscriptionStatusPending {
		return syncPendingSubscription(&sub)
	}

	// Path 2: Active subscription-mode with StripeSubscriptionID
	if sub.PaymentMode == billing_model.PaymentModeSubscription && sub.StripeSubscriptionID != nil {
		return syncActiveSubscription(&sub)
	}

	return billing_entity.Subscription{}, errors.New("this subscription cannot be synced")
}

// syncPendingSubscription syncs a pending subscription by checking the checkout session status.
func syncPendingSubscription(sub *billing_entity.Subscription) (billing_entity.Subscription, error) {
	if sub.PaymentExternalID == nil {
		return *sub, errors.New("pending subscription has no payment external ID")
	}

	sessionStatus, err := payment.ActiveProvider.GetCheckoutSessionStatus(*sub.PaymentExternalID)
	if err != nil {
		return *sub, fmt.Errorf("failed to get checkout session status: %w", err)
	}

	if sessionStatus.PaymentStatus == "paid" {
		// Activate the subscription
		now := time.Now()
		sub.Status = billing_model.SubscriptionStatusActive
		sub.StartsAt = now
		if sub.Plan != nil {
			sub.ExpiresAt = now.AddDate(0, 0, sub.Plan.DurationDays)
		}
		if sessionStatus.StripeSubscriptionID != "" {
			sub.StripeSubscriptionID = &sessionStatus.StripeSubscriptionID
		}

		if err := database.DB.Save(sub).Error; err != nil {
			return *sub, fmt.Errorf("failed to activate subscription: %w", err)
		}

		// Persist the Stripe Customer ID on the user
		if sessionStatus.CustomerID != "" {
			database.DB.Model(&user_entity.User{}).
				Where("id = ? AND (stripe_customer_id IS NULL OR stripe_customer_id = '')", sub.UserID).
				Update("stripe_customer_id", sessionStatus.CustomerID)
		}

		invalidateForSubscription(sub)
		return *sub, nil
	}

	if sessionStatus.SessionStatus == "expired" {
		// Checkout session expired — mark subscription as cancelled
		now := time.Now()
		sub.Status = billing_model.SubscriptionStatusCancelled
		sub.CancelledAt = &now
		if err := database.DB.Save(sub).Error; err != nil {
			return *sub, fmt.Errorf("failed to cancel expired subscription: %w", err)
		}
		return *sub, nil
	}

	// Session still open or unpaid — return as-is
	return *sub, nil
}

// syncActiveSubscription syncs an active subscription-mode subscription via the Stripe subscription API.
func syncActiveSubscription(sub *billing_entity.Subscription) (billing_entity.Subscription, error) {
	details, err := payment.ActiveProvider.GetSubscriptionDetails(*sub.StripeSubscriptionID)
	if err != nil {
		return *sub, fmt.Errorf("failed to get subscription details from provider: %w", err)
	}

	if details.Status == "active" || details.Status == "trialing" {
		sub.ExpiresAt = details.CurrentPeriodEnd
	} else {
		// If past_due, unpaid, paused, etc., the user hasn't successfully paid for the current period.
		// We ensure ExpiresAt is not in the future so IsActive() returns false.
		// We don't set CancelledAt so that a future invoice.paid can still renew it.
		now := time.Now()
		if sub.ExpiresAt.After(now) {
			sub.ExpiresAt = now.Add(-1 * time.Second)
		}
	}

	sub.CancelAtPeriodEnd = details.CancelAtPeriodEnd

	if details.Status == "canceled" && sub.CancelledAt == nil {
		now := time.Now()
		sub.CancelledAt = &now
		sub.Status = billing_model.SubscriptionStatusCancelled
	}

	if err := database.DB.Save(sub).Error; err != nil {
		return *sub, fmt.Errorf("failed to save synced subscription: %w", err)
	}

	invalidateForSubscription(sub)

	return *sub, nil
}

// GetActiveSubscriptions returns all active subscriptions for a scope.
func GetActiveSubscriptions(scope billing_model.Scope, userID *uuid.UUID, workspaceID *uuid.UUID) ([]billing_entity.Subscription, error) {
	now := time.Now()
	query := database.DB.
		Preload("Plan").
		Where("scope = ? AND starts_at <= ? AND expires_at > ? AND cancelled_at IS NULL AND status = 'active'", scope, now, now)

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

// RetrySubscription returns a URL a user can visit to pay a past due or unpaid subscription.
func RetrySubscription(subscriptionID uuid.UUID, userID uuid.UUID, workspaceID *uuid.UUID) (string, error) {
	var sub billing_entity.Subscription
	if err := database.DB.First(&sub, subscriptionID).Error; err != nil {
		return "", errors.New("subscription not found")
	}

	// Ownership check: workspace-scoped or user-scoped
	if workspaceID != nil {
		if sub.WorkspaceID == nil || *sub.WorkspaceID != *workspaceID {
			return "", errors.New("unauthorized: subscription does not belong to this workspace")
		}
	} else if sub.UserID != userID {
		return "", errors.New("unauthorized: you can only retry your own subscriptions")
	}

	if sub.PaymentMode != billing_model.PaymentModeSubscription {
		return "", errors.New("only recurring subscriptions can be retried")
	}

	if sub.StripeSubscriptionID == nil || *sub.StripeSubscriptionID == "" {
		return "", errors.New("this subscription has no associated payment provider subscription")
	}

	if payment.ActiveProvider == nil {
		return "", errors.New("payment provider is not configured")
	}

	return payment.ActiveProvider.GetSubscriptionRetryURL(*sub.StripeSubscriptionID)
}

// ResumeCheckout returns a live checkout URL for a pending subscription.
// If the original Stripe checkout session is still open, its hosted URL is
// returned as-is. If it expired, a fresh checkout session is created for the
// same plan (reusing the currency from the original session), the pending
// subscription's external ID is updated in place, and the new URL is returned.
func ResumeCheckout(subscriptionID uuid.UUID, userID uuid.UUID, workspaceID *uuid.UUID, successURL, cancelURL string) (billing_model.CheckoutResponse, error) {
	if payment.ActiveProvider == nil {
		return billing_model.CheckoutResponse{}, errors.New("payment provider is not configured")
	}

	var sub billing_entity.Subscription
	if err := database.DB.Preload("Plan").First(&sub, subscriptionID).Error; err != nil {
		return billing_model.CheckoutResponse{}, errors.New("subscription not found")
	}

	// Ownership check: workspace-scoped or user-scoped
	if workspaceID != nil {
		if sub.WorkspaceID == nil || *sub.WorkspaceID != *workspaceID {
			return billing_model.CheckoutResponse{}, errors.New("unauthorized: subscription does not belong to this workspace")
		}
	} else if sub.UserID != userID {
		return billing_model.CheckoutResponse{}, errors.New("unauthorized: you can only resume your own checkouts")
	}

	if sub.Status != billing_model.SubscriptionStatusPending {
		return billing_model.CheckoutResponse{}, errors.New("only pending subscriptions can be resumed")
	}
	if sub.PaymentExternalID == nil || *sub.PaymentExternalID == "" {
		return billing_model.CheckoutResponse{}, errors.New("pending subscription has no checkout session")
	}

	status, err := payment.ActiveProvider.GetCheckoutSessionStatus(*sub.PaymentExternalID)
	if err != nil {
		return billing_model.CheckoutResponse{}, fmt.Errorf("failed to get checkout session status: %w", err)
	}

	// Already paid (webhook may be lagging) — activate it and report so the client refreshes.
	if status.PaymentStatus == "paid" {
		if _, aerr := syncPendingSubscription(&sub); aerr != nil {
			return billing_model.CheckoutResponse{}, aerr
		}
		return billing_model.CheckoutResponse{}, errors.New("payment already completed — refresh to see your active subscription")
	}

	// Session still open — return the existing hosted checkout URL.
	if status.SessionStatus == "open" && status.URL != "" {
		return billing_model.CheckoutResponse{CheckoutURL: status.URL, ExternalID: *sub.PaymentExternalID}, nil
	}

	// Session expired (or otherwise unusable) — regenerate for the same plan.
	return regenerateCheckout(&sub, status.Currency, successURL, cancelURL)
}

// regenerateCheckout creates a fresh checkout session for a pending subscription
// whose original session expired, updating the pending row in place.
func regenerateCheckout(sub *billing_entity.Subscription, currency, successURL, cancelURL string) (billing_model.CheckoutResponse, error) {
	var plan billing_entity.Plan
	if err := database.DB.First(&plan, sub.PlanID).Error; err != nil {
		return billing_model.CheckoutResponse{}, errors.New("plan not found")
	}
	if !plan.Active {
		return billing_model.CheckoutResponse{}, errors.New("this plan is no longer available for purchase")
	}

	// Resolve the plan price by the original session's currency, falling back to
	// the plan's default price when that currency is no longer configured.
	var planPrice billing_entity.PlanPrice
	priceQuery := database.DB.Where("plan_id = ?", sub.PlanID)
	if currency != "" {
		priceQuery = priceQuery.Where("currency = ?", currency)
	} else {
		priceQuery = priceQuery.Where("is_default = true")
	}
	if err := priceQuery.First(&planPrice).Error; err != nil {
		if err := database.DB.Where("plan_id = ? AND is_default = true", sub.PlanID).First(&planPrice).Error; err != nil {
			return billing_model.CheckoutResponse{}, errors.New("no price configured for this plan")
		}
	}

	var user user_entity.User
	if err := database.DB.First(&user, sub.UserID).Error; err != nil {
		return billing_model.CheckoutResponse{}, errors.New("user not found")
	}

	checkoutURL, externalID, err := payment.ActiveProvider.CreateCheckoutSession(
		plan, planPrice, sub.PaymentMode, sub.UserID, user.Email, user.StripeCustomerID,
		sub.Scope, sub.WorkspaceID, successURL, cancelURL,
	)
	if err != nil {
		return billing_model.CheckoutResponse{}, fmt.Errorf("unable to create checkout session: %w", err)
	}

	// Point the existing pending subscription at the new session.
	if err := database.DB.Model(sub).Update("payment_external_id", externalID).Error; err != nil {
		return billing_model.CheckoutResponse{}, fmt.Errorf("failed to update pending subscription: %w", err)
	}

	return billing_model.CheckoutResponse{CheckoutURL: checkoutURL, ExternalID: externalID}, nil
}

// ListPayments returns a page of payments read live from the payment provider.
// User scope reads the authenticated user's payments; workspace scope reads
// payments across the Stripe customers that hold a subscription for the workspace.
func ListPayments(userID uuid.UUID, workspaceID *uuid.UUID, limit int, cursor string) (billing_model.PaymentListResponse, error) {
	if payment.ActiveProvider == nil {
		return billing_model.PaymentListResponse{}, errors.New("payment provider is not configured")
	}

	customerIDs, err := resolveCustomerIDs(userID, workspaceID)
	if err != nil {
		return billing_model.PaymentListResponse{}, err
	}
	if len(customerIDs) == 0 {
		return billing_model.PaymentListResponse{Data: []billing_model.Payment{}}, nil
	}

	payments, hasMore, nextCursor, err := payment.ActiveProvider.ListPayments(customerIDs, limit, cursor)
	if err != nil {
		return billing_model.PaymentListResponse{}, err
	}
	if payments == nil {
		payments = []billing_model.Payment{}
	}

	return billing_model.PaymentListResponse{
		Data:       payments,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

// resolveCustomerIDs returns the Stripe customer IDs to read payments for.
// User scope: the user's own customer. Workspace scope: distinct customers of
// every user who holds a subscription for the workspace.
func resolveCustomerIDs(userID uuid.UUID, workspaceID *uuid.UUID) ([]string, error) {
	if workspaceID == nil {
		var user user_entity.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			return nil, errors.New("user not found")
		}
		if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
			return nil, nil
		}
		return []string{*user.StripeCustomerID}, nil
	}

	var userIDs []uuid.UUID
	if err := database.DB.
		Model(&billing_entity.Subscription{}).
		Where("workspace_id = ?", *workspaceID).
		Distinct().
		Pluck("user_id", &userIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to resolve workspace subscribers: %w", err)
	}
	if len(userIDs) == 0 {
		return nil, nil
	}

	var ids []string
	if err := database.DB.
		Model(&user_entity.User{}).
		Where("id IN ? AND stripe_customer_id IS NOT NULL AND stripe_customer_id != ''", userIDs).
		Distinct().
		Pluck("stripe_customer_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("failed to resolve workspace customers: %w", err)
	}
	return ids, nil
}
