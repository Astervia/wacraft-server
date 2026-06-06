package billing_handler

import (
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	billing_service "github.com/Astervia/wacraft-server/src/billing/service"
	"github.com/Astervia/wacraft-server/src/billing/service/payment"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetPayments returns a cursor-paginated page of payments read live from the
// payment provider.
// Without X-Workspace-ID: the authenticated user's payments.
// With X-Workspace-ID: payments across the workspace's subscribers (requires billing.read policy).
//
//	@Summary		List payments
//	@Description	Returns a page of payments read live from the payment provider (e.g. Stripe). Pagination is cursor-based: pass the returned next_cursor to fetch the next page. Without X-Workspace-ID returns the user's payments; with X-Workspace-ID returns workspace payments (requires billing.read policy).
//	@Tags			Billing Payment
//	@Accept			json
//	@Produce		json
//	@Param			payment	query		billing_model.PaymentQuery			true	"Pagination parameters"
//	@Success		200		{object}	billing_model.PaymentListResponse	"Page of payments"
//	@Failure		400		{object}	common_model.DescriptiveError		"Invalid query parameters"
//	@Failure		403		{object}	common_model.DescriptiveError		"Insufficient permissions"
//	@Failure		500		{object}	common_model.DescriptiveError		"Internal server error"
//	@Failure		503		{object}	common_model.DescriptiveError		"Payment provider not configured"
//	@Security		ApiKeyAuth
//	@Router			/billing/payment/ [get]
func GetPayments(c *fiber.Ctx) error {
	if payment.ActiveProvider == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(
			common_model.NewApiError("payment provider is not configured", nil, "billing").Send(),
		)
	}

	user := workspace_middleware.GetUser(c)

	query := new(billing_model.PaymentQuery)
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

	var workspaceID *uuid.UUID
	if workspace := workspace_middleware.GetWorkspace(c); workspace != nil {
		wsID, err := requireWorkspacePolicy(c, workspace_model.PolicyBillingRead)
		if err != nil {
			return err
		}
		workspaceID = wsID
	}

	resp, err := billing_service.ListPayments(user.ID, workspaceID, query.Limit, query.Cursor)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to list payments", err, "billing_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
