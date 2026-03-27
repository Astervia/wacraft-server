package billing_handler

import (
	"log"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	billing_service "github.com/Astervia/wacraft-server/src/billing/service"
	"github.com/Astervia/wacraft-server/src/billing/service/payment"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// requireWorkspacePolicy checks that the workspace user has the given policy
// and returns the workspace ID. Returns a non-nil error response if the check fails.
func requireWorkspacePolicy(c *fiber.Ctx, policy workspace_model.Policy) (*uuid.UUID, error) {
	if !workspace_middleware.HasPolicy(c, policy) {
		return nil, c.Status(fiber.StatusForbidden).JSON(
			common_model.NewApiError("insufficient permissions", nil, "handler").Send(),
		)
	}
	workspace := workspace_middleware.GetWorkspace(c)
	return &workspace.ID, nil
}

// GetSubscriptions returns paginated subscriptions.
// Without X-Workspace-ID: returns the authenticated user's subscriptions.
// With X-Workspace-ID: returns workspace subscriptions (requires billing.read policy).
//
//	@Summary		Retrieve subscriptions
//	@Description	Returns paginated subscriptions. Without X-Workspace-ID returns user subscriptions. With X-Workspace-ID returns workspace subscriptions (requires billing.read policy).
//	@Tags			Billing Subscription
//	@Accept			json
//	@Produce		json
//	@Param			subscription	query		billing_model.SubscriptionQueryPaginated	true	"Pagination and query parameters"
//	@Success		200				{array}		billing_entity.Subscription					"List of subscriptions"
//	@Failure		400				{object}	common_model.DescriptiveError				"Invalid query parameters"
//	@Failure		403				{object}	common_model.DescriptiveError				"Insufficient permissions"
//	@Failure		500				{object}	common_model.DescriptiveError				"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/billing/subscription/ [get]
func GetSubscriptions(c *fiber.Ctx) error {
	user := workspace_middleware.GetUser(c)

	query := new(billing_model.SubscriptionQueryPaginated)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	var filter billing_entity.Subscription

	if workspace := workspace_middleware.GetWorkspace(c); workspace != nil {
		// Workspace-scoped: require billing.read policy
		wsID, err := requireWorkspacePolicy(c, workspace_model.PolicyBillingRead)
		if err != nil {
			return err
		}
		filter.WorkspaceID = wsID
	} else {
		// User-scoped: return own subscriptions
		filter.UserID = user.ID
	}

	subs, err := repository.GetPaginated(
		filter,
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB.Preload("Plan"),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get subscriptions", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(subs)
}

// Checkout initiates a payment flow for purchasing a plan.
//
//	@Summary		Initiate checkout
//	@Description	Creates a payment checkout session for purchasing a plan. Returns a URL to redirect the user to the payment provider.
//	@Tags			Billing Subscription
//	@Accept			json
//	@Produce		json
//	@Param			checkout	body		billing_model.CheckoutRequest	true	"Checkout data"
//	@Success		200			{object}	billing_model.CheckoutResponse	"Checkout session created"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		404			{object}	common_model.DescriptiveError	"Plan not found"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Failure		503			{object}	common_model.DescriptiveError	"Payment provider not configured"
//	@Security		ApiKeyAuth
//	@Router			/billing/subscription/checkout [post]
func Checkout(c *fiber.Ctx) error {
	if payment.ActiveProvider == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(
			common_model.NewApiError("payment provider is not configured", nil, "billing").Send(),
		)
	}

	user := workspace_middleware.GetUser(c)

	body := new(billing_model.CheckoutRequest)
	if err := c.BodyParser(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	// When workspace header is present, auto-fill scope and workspace ID
	if workspace := workspace_middleware.GetWorkspace(c); workspace != nil {
		wsID, err := requireWorkspacePolicy(c, workspace_model.PolicyBillingManage)
		if err != nil {
			return err
		}
		body.Scope = billing_model.ScopeWorkspace
		body.WorkspaceID = wsID
	}

	if body.Scope == billing_model.ScopeWorkspace && body.WorkspaceID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("workspace_id is required for workspace-scoped plans", nil, "handler").Send(),
		)
	}

	// Fetch the plan
	var plan billing_entity.Plan
	if err := database.DB.First(&plan, body.PlanID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("plan not found", err, "handler").Send(),
		)
	}

	if !plan.Active {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("this plan is not available for purchase", nil, "handler").Send(),
		)
	}

	// Resolve the plan price: by requested currency or by the default price.
	var planPrice billing_entity.PlanPrice
	priceQuery := database.DB.Where("plan_id = ?", body.PlanID)
	if body.Currency != "" {
		priceQuery = priceQuery.Where("currency = ?", body.Currency)
	} else {
		priceQuery = priceQuery.Where("is_default = true")
	}
	if err := priceQuery.First(&planPrice).Error; err != nil {
		msg := "no default price configured for this plan"
		if body.Currency != "" {
			msg = "no price configured for currency: " + body.Currency
		}
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError(msg, err, "handler").Send(),
		)
	}

	// Default to payment (one-time) mode if not specified
	paymentMode := body.PaymentMode
	if paymentMode == "" {
		paymentMode = billing_model.PaymentModePayment
	}

	checkoutURL, externalID, err := payment.ActiveProvider.CreateCheckoutSession(
		plan, planPrice, paymentMode, user.ID, user.Email, user.StripeCustomerID,
		body.Scope, body.WorkspaceID, body.SuccessURL, body.CancelURL,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create checkout session", err, "payment").Send(),
		)
	}

	// Pre-create a pending subscription so a local record exists for sync recovery
	// if the webhook fails. Errors here are non-fatal — the webhook can still create
	// the subscription as fallback.
	if _, err := billing_service.CreatePendingSubscription(
		body.PlanID, body.Scope, user.ID, body.WorkspaceID,
		payment.ActiveProvider.Name(), externalID, paymentMode,
	); err != nil {
		log.Printf("warning: failed to create pending subscription: %v", err)
	}

	return c.Status(fiber.StatusOK).JSON(billing_model.CheckoutResponse{
		CheckoutURL: checkoutURL,
		ExternalID:  externalID,
	})
}

// CreateManualSubscription creates a subscription manually (admin only).
//
//	@Summary		Create manual subscription
//	@Description	Creates a subscription manually for custom deals or enterprise plans. Requires billing.admin policy.
//	@Tags			Billing Subscription
//	@Accept			json
//	@Produce		json
//	@Param			subscription	body		billing_model.CreateManualSubscription	true	"Subscription data"
//	@Success		201				{object}	billing_entity.Subscription				"Created subscription"
//	@Failure		400				{object}	common_model.DescriptiveError			"Invalid request body"
//	@Failure		500				{object}	common_model.DescriptiveError			"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/subscription/manual [post]
func CreateManualSubscription(c *fiber.Ctx) error {
	body := new(billing_model.CreateManualSubscription)
	if err := c.BodyParser(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	sub, err := billing_service.CreateManualSubscription(*body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create subscription", err, "billing_service").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(sub)
}

// ReactivateSubscription reverses a pending cancellation, re-enabling auto-renewal.
//
//	@Summary		Reactivate a subscription
//	@Description	Reverses a pending cancellation for a recurring subscription, re-enabling auto-renewal. Only works when cancel_at_period_end is true and cancelled_at is not set.
//	@Tags			Billing Subscription
//	@Accept			json
//	@Produce		json
//	@Param			id	query		string							true	"Subscription ID"
//	@Success		204	{string}	string							"No content"
//	@Failure		400	{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500	{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/billing/subscription/reactivate [post]
func ReactivateSubscription(c *fiber.Ctx) error {
	user := workspace_middleware.GetUser(c)

	id := new(common_model.RequiredID)
	if err := c.QueryParser(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	var workspaceID *uuid.UUID
	if workspace := workspace_middleware.GetWorkspace(c); workspace != nil {
		wsID, err := requireWorkspacePolicy(c, workspace_model.PolicyBillingManage)
		if err != nil {
			return err
		}
		workspaceID = wsID
	}

	if err := billing_service.ReactivateSubscription(id.ID, user.ID, workspaceID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to reactivate subscription", err, "billing_service").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// SyncSubscription fetches the current subscription state from the payment provider
// and reconciles the local DB record.
//
//	@Summary		Sync subscription with payment provider
//	@Description	Fetches the current subscription state from the payment provider (e.g. Stripe) and updates the local record. Only works for subscription-mode subscriptions.
//	@Tags			Billing Subscription
//	@Accept			json
//	@Produce		json
//	@Param			id	query		string							true	"Subscription ID"
//	@Success		200	{object}	billing_entity.Subscription		"Updated subscription"
//	@Failure		400	{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500	{object}	common_model.DescriptiveError	"Internal server error"
//	@Failure		503	{object}	common_model.DescriptiveError	"Payment provider not configured"
//	@Security		ApiKeyAuth
//	@Router			/billing/subscription/sync [post]
func SyncSubscription(c *fiber.Ctx) error {
	if payment.ActiveProvider == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(
			common_model.NewApiError("payment provider is not configured", nil, "billing").Send(),
		)
	}

	user := workspace_middleware.GetUser(c)

	id := new(common_model.RequiredID)
	if err := c.QueryParser(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	var workspaceID *uuid.UUID
	if workspace := workspace_middleware.GetWorkspace(c); workspace != nil {
		wsID, err := requireWorkspacePolicy(c, workspace_model.PolicyBillingRead)
		if err != nil {
			return err
		}
		workspaceID = wsID
	}

	sub, err := billing_service.SyncSubscription(id.ID, user.ID, workspaceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to sync subscription", err, "billing_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(sub)
}

// CancelSubscription cancels an active subscription.
//
//	@Summary		Cancel a subscription
//	@Description	Cancels an active subscription. Users can only cancel their own subscriptions.
//	@Tags			Billing Subscription
//	@Accept			json
//	@Produce		json
//	@Param			id	query		string							true	"Subscription ID"
//	@Success		204	{string}	string							"No content"
//	@Failure		400	{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500	{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/billing/subscription/ [delete]
func CancelSubscription(c *fiber.Ctx) error {
	user := workspace_middleware.GetUser(c)

	id := new(common_model.RequiredID)
	if err := c.QueryParser(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	var workspaceID *uuid.UUID
	if workspace := workspace_middleware.GetWorkspace(c); workspace != nil {
		wsID, err := requireWorkspacePolicy(c, workspace_model.PolicyBillingManage)
		if err != nil {
			return err
		}
		workspaceID = wsID
	}

	if err := billing_service.CancelSubscription(id.ID, user.ID, workspaceID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to cancel subscription", err, "billing_service").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RetrySubscription returns a URL to retry payment for a past due subscription.
//
//	@Summary		Retry subscription payment
//	@Description	Returns a URL to the payment provider where the user can pay an outstanding invoice for a past due subscription. Without X-Workspace-ID it operates on a user subscription. With X-Workspace-ID it operates on a workspace subscription (requires billing.manage policy).
//	@Tags			Billing Subscription
//	@Accept			json
//	@Produce		json
//	@Param			id	query		string							true	"Subscription ID"
//	@Success		200	{string}	string				"Retry URL"
//	@Failure		400	{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500	{object}	common_model.DescriptiveError	"Internal server error"
//	@Failure		503	{object}	common_model.DescriptiveError	"Payment provider not configured"
//	@Security		ApiKeyAuth
//	@Router			/billing/subscription/retry [post]
func RetrySubscription(c *fiber.Ctx) error {
	if payment.ActiveProvider == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(
			common_model.NewApiError("payment provider is not configured", nil, "billing").Send(),
		)
	}

	user := workspace_middleware.GetUser(c)

	id := new(common_model.RequiredID)
	if err := c.QueryParser(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	var workspaceID *uuid.UUID
	if workspace := workspace_middleware.GetWorkspace(c); workspace != nil {
		wsID, err := requireWorkspacePolicy(c, workspace_model.PolicyBillingManage)
		if err != nil {
			return err
		}
		workspaceID = wsID
	}

	url, err := billing_service.RetrySubscription(id.ID, user.ID, workspaceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get subscription retry url", err, "billing_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).SendString(url)
}
