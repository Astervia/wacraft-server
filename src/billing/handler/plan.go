package billing_handler

import (
	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// GetPlans returns a paginated list of billing plans.
//
//	@Summary		Retrieve billing plans
//	@Description	Returns a paginated list of billing plans based on optional filters.
//	@Tags			Billing Plan
//	@Accept			json
//	@Produce		json
//	@Param			plan	query		billing_model.PlanQueryPaginated	true	"Pagination and query parameters"
//	@Success		200		{array}		billing_entity.Plan					"List of plans"
//	@Failure		400		{object}	common_model.DescriptiveError		"Invalid query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError		"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/billing/plan/ [get]
func GetPlans(c *fiber.Ctx) error {
	query := new(billing_model.PlanQueryPaginated)
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

	plans, err := repository.GetPaginated(
		billing_entity.Plan{},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get plans", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(plans)
}

// CreatePlan creates a new billing plan (admin only).
//
//	@Summary		Create a new billing plan
//	@Description	Creates a new billing plan using the provided data. Requires billing.admin policy.
//	@Tags			Billing Plan
//	@Accept			json
//	@Produce		json
//	@Param			plan	body		billing_model.CreatePlan			true	"Plan data"
//	@Success		201		{object}	billing_entity.Plan				"Created plan"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/plan/ [post]
func CreatePlan(c *fiber.Ctx) error {
	body := new(billing_model.CreatePlan)
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

	plan := billing_entity.Plan{
		Name:            body.Name,
		Slug:            body.Slug,
		Description:     body.Description,
		ThroughputLimit: body.ThroughputLimit,
		WindowSeconds:   body.WindowSeconds,
		DurationDays:    body.DurationDays,
		PriceCents:      body.PriceCents,
		Currency:        body.Currency,
		IsDefault:       body.IsDefault,
		IsCustom:        body.IsCustom,
		Active:          body.Active,
	}

	result, err := repository.Create(plan, database.DB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create plan", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// UpdatePlan updates an existing billing plan (admin only).
//
//	@Summary		Update a billing plan
//	@Description	Updates an existing billing plan. Requires billing.admin policy.
//	@Tags			Billing Plan
//	@Accept			json
//	@Produce		json
//	@Param			id		query		string							true	"Plan ID"
//	@Param			plan	body		billing_model.UpdatePlan			true	"Updated plan data"
//	@Success		200		{object}	billing_entity.Plan				"Updated plan"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/plan/ [put]
func UpdatePlan(c *fiber.Ctx) error {
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

	body := new(billing_model.UpdatePlan)
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

	// Build update map from non-nil pointer fields to handle zero values correctly
	updates := make(map[string]any)
	if body.Name != nil {
		updates["name"] = *body.Name
	}
	if body.Slug != nil {
		updates["slug"] = *body.Slug
	}
	if body.Description != nil {
		updates["description"] = *body.Description
	}
	if body.ThroughputLimit != nil {
		updates["throughput_limit"] = *body.ThroughputLimit
	}
	if body.WindowSeconds != nil {
		updates["window_seconds"] = *body.WindowSeconds
	}
	if body.DurationDays != nil {
		updates["duration_days"] = *body.DurationDays
	}
	if body.PriceCents != nil {
		updates["price_cents"] = *body.PriceCents
	}
	if body.Currency != nil {
		updates["currency"] = *body.Currency
	}
	if body.IsDefault != nil {
		updates["is_default"] = *body.IsDefault
	}
	if body.IsCustom != nil {
		updates["is_custom"] = *body.IsCustom
	}
	if body.Active != nil {
		updates["active"] = *body.Active
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusOK).JSON(nil)
	}

	plan, err := repository.Updates[billing_entity.Plan](
		updates,
		&billing_entity.Plan{Audit: common_model.Audit{ID: id.ID}},
		database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update plan", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(plan)
}

// DeletePlan deletes a billing plan (admin only).
//
//	@Summary		Delete a billing plan
//	@Description	Deletes a billing plan by ID. Plans with active subscriptions cannot be deleted. Requires billing.admin policy.
//	@Tags			Billing Plan
//	@Accept			json
//	@Produce		json
//	@Param			id	query		string							true	"Plan ID"
//	@Success		204	{string}	string							"No content"
//	@Failure		400	{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500	{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/plan/ [delete]
func DeletePlan(c *fiber.Ctx) error {
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

	if err := repository.DeleteByID[billing_entity.Plan](id.ID, database.DB); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete plan", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
