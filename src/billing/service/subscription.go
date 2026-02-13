package billing_service

import (
	"errors"
	"time"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
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
func ActivateSubscription(planID uuid.UUID, scope billing_model.Scope, userID uuid.UUID, workspaceID *uuid.UUID, provider string, externalID string) (billing_entity.Subscription, error) {
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
	}

	if err := database.DB.Create(&sub).Error; err != nil {
		return sub, err
	}

	invalidateForSubscription(&sub)

	return sub, nil
}

// CancelSubscription marks a subscription as cancelled.
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

	now := time.Now()
	sub.CancelledAt = &now
	if err := database.DB.Save(&sub).Error; err != nil {
		return err
	}

	invalidateForSubscription(&sub)

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
