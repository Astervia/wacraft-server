package billing_handler

import (
	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetPlanPrices returns all prices for a specific plan.
//
//	@Summary		List plan prices
//	@Description	Returns all currency-specific prices for a billing plan.
//	@Tags			Billing Plan Price
//	@Accept			json
//	@Produce		json
//	@Param			plan_id	path		string									true	"Plan ID"
//	@Param			query	query		billing_model.PlanPriceQueryPaginated	true	"Pagination parameters"
//	@Success		200		{array}		billing_entity.PlanPrice				"List of plan prices"
//	@Failure		400		{object}	common_model.DescriptiveError			"Invalid parameters"
//	@Failure		500		{object}	common_model.DescriptiveError			"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/billing/plan/{plan_id}/price/ [get]
func GetPlanPrices(c *fiber.Ctx) error {
	planID, err := uuid.Parse(c.Params("plan_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("invalid plan_id", err, "handler").Send(),
		)
	}

	query := new(billing_model.PlanPriceQueryPaginated)
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

	prices, err := repository.GetPaginated(
		billing_entity.PlanPrice{PlanID: planID},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get plan prices", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(prices)
}

// CreatePlanPrice adds a currency-specific price to a plan (admin only).
//
//	@Summary		Create a plan price
//	@Description	Adds a new currency-specific price to a billing plan. If IsDefault is true, the previous default for that plan is unset. Requires billing.admin policy.
//	@Tags			Billing Plan Price
//	@Accept			json
//	@Produce		json
//	@Param			plan_id	path		string							true	"Plan ID"
//	@Param			price	body		billing_model.CreatePlanPrice	true	"Price data"
//	@Success		201		{object}	billing_entity.PlanPrice		"Created plan price"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/plan/{plan_id}/price/ [post]
func CreatePlanPrice(c *fiber.Ctx) error {
	planID, err := uuid.Parse(c.Params("plan_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("invalid plan_id", err, "handler").Send(),
		)
	}

	body := new(billing_model.CreatePlanPrice)
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

	// If this price is the default, unset any existing default for this plan first.
	if body.IsDefault {
		if err := database.DB.
			Model(&billing_entity.PlanPrice{}).
			Where("plan_id = ? AND is_default = true", planID).
			Update("is_default", false).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("unable to unset previous default price", err, "repository").Send(),
			)
		}
	}

	planPrice := billing_entity.PlanPrice{
		PlanID:     planID,
		Currency:   body.Currency,
		PriceCents: body.PriceCents,
		IsDefault:  body.IsDefault,
	}

	result, err := repository.Create(planPrice, database.DB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create plan price", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// UpdatePlanPrice updates a plan price entry (admin only).
//
//	@Summary		Update a plan price
//	@Description	Updates an existing plan price. If IsDefault is set to true, the previous default for that plan is unset. Requires billing.admin policy.
//	@Tags			Billing Plan Price
//	@Accept			json
//	@Produce		json
//	@Param			plan_id	path		string							true	"Plan ID"
//	@Param			id		query		string							true	"Plan Price ID"
//	@Param			price	body		billing_model.UpdatePlanPrice	true	"Updated price data"
//	@Success		200		{object}	billing_entity.PlanPrice		"Updated plan price"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/plan/{plan_id}/price/ [put]
func UpdatePlanPrice(c *fiber.Ctx) error {
	planID, err := uuid.Parse(c.Params("plan_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("invalid plan_id", err, "handler").Send(),
		)
	}

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

	body := new(billing_model.UpdatePlanPrice)
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

	updates := make(map[string]any)
	if body.PriceCents != nil {
		updates["price_cents"] = *body.PriceCents
	}
	if body.IsDefault != nil {
		updates["is_default"] = *body.IsDefault
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusOK).JSON(nil)
	}

	// If promoting to default, unset the previous default for this plan first.
	if body.IsDefault != nil && *body.IsDefault {
		if err := database.DB.
			Model(&billing_entity.PlanPrice{}).
			Where("plan_id = ? AND is_default = true AND id != ?", planID, id.ID).
			Update("is_default", false).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("unable to unset previous default price", err, "repository").Send(),
			)
		}
	}

	planPrice, err := repository.Updates[billing_entity.PlanPrice](
		updates,
		&billing_entity.PlanPrice{Audit: common_model.Audit{ID: id.ID}},
		database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update plan price", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(planPrice)
}

// DeletePlanPrice removes a currency-specific price from a plan (admin only).
//
//	@Summary		Delete a plan price
//	@Description	Removes a currency-specific price from a billing plan. Requires billing.admin policy.
//	@Tags			Billing Plan Price
//	@Accept			json
//	@Produce		json
//	@Param			plan_id	path		string							true	"Plan ID"
//	@Param			id		query		string							true	"Plan Price ID"
//	@Success		204		{string}	string							"No content"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/plan/{plan_id}/price/ [delete]
func DeletePlanPrice(c *fiber.Ctx) error {
	if _, err := uuid.Parse(c.Params("plan_id")); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("invalid plan_id", err, "handler").Send(),
		)
	}

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

	if err := repository.DeleteByID[billing_entity.PlanPrice](id.ID, database.DB); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete plan price", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
