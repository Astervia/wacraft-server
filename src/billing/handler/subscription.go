package billing_handler

import (
	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	billing_service "github.com/Astervia/wacraft-server/src/billing/service"
	"github.com/Astervia/wacraft-server/src/billing/service/payment"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// GetSubscriptions returns paginated subscriptions for the current user.
// If a workspace is in context, filters by that workspace too.
//
//	@Summary		Retrieve subscriptions
//	@Description	Returns paginated subscriptions for the authenticated user. If X-Workspace-ID is provided, filters by that workspace.
//	@Tags			Billing Subscription
//	@Accept			json
//	@Produce		json
//	@Param			subscription	query		billing_model.SubscriptionQueryPaginated	true	"Pagination and query parameters"
//	@Success		200				{array}		billing_entity.Subscription					"List of subscriptions"
//	@Failure		400				{object}	common_model.DescriptiveError				"Invalid query parameters"
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

	filter := billing_entity.Subscription{UserID: user.ID}

	// Optionally filter by workspace if present
	workspace := workspace_middleware.GetWorkspace(c)
	if workspace != nil {
		filter.WorkspaceID = &workspace.ID
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

	// Default to payment (one-time) mode if not specified
	paymentMode := body.PaymentMode
	if paymentMode == "" {
		paymentMode = billing_model.PaymentModePayment
	}

	checkoutURL, externalID, err := payment.ActiveProvider.CreateCheckoutSession(
		plan, paymentMode, user.ID, user.Email, user.StripeCustomerID,
		body.Scope, body.WorkspaceID, body.SuccessURL, body.CancelURL,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create checkout session", err, "payment").Send(),
		)
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

	if err := billing_service.ReactivateSubscription(id.ID, user.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to reactivate subscription", err, "billing_service").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
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

	if err := billing_service.CancelSubscription(id.ID, user.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to cancel subscription", err, "billing_service").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
