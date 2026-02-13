package billing_handler

import (
	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	billing_service "github.com/Astervia/wacraft-server/src/billing/service"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// GetEndpointWeights returns a paginated list of endpoint weights.
//
//	@Summary		Retrieve endpoint weights
//	@Description	Returns a paginated list of endpoint weight configurations. Requires billing.admin policy.
//	@Tags			Billing Endpoint Weight
//	@Accept			json
//	@Produce		json
//	@Param			weight	query		billing_model.EndpointWeightQueryPaginated	true	"Pagination and query parameters"
//	@Success		200		{array}		billing_entity.EndpointWeight				"List of endpoint weights"
//	@Failure		400		{object}	common_model.DescriptiveError				"Invalid query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError				"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/endpoint-weight/ [get]
func GetEndpointWeights(c *fiber.Ctx) error {
	query := new(billing_model.EndpointWeightQueryPaginated)
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

	weights, err := repository.GetPaginated(
		billing_entity.EndpointWeight{},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get endpoint weights", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(weights)
}

// CreateEndpointWeight creates or updates an endpoint weight (admin only).
//
//	@Summary		Create an endpoint weight
//	@Description	Creates a custom endpoint weight configuration. Requires billing.admin policy.
//	@Tags			Billing Endpoint Weight
//	@Accept			json
//	@Produce		json
//	@Param			weight	body		billing_model.CreateEndpointWeight	true	"Endpoint weight data"
//	@Success		201		{object}	billing_entity.EndpointWeight	"Created endpoint weight"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/endpoint-weight/ [post]
func CreateEndpointWeight(c *fiber.Ctx) error {
	body := new(billing_model.CreateEndpointWeight)
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

	weight := billing_entity.EndpointWeight{
		Method:      body.Method,
		PathPattern: body.PathPattern,
		Weight:      body.Weight,
		Description: body.Description,
	}

	result, err := repository.Create(weight, database.DB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create endpoint weight", err, "repository").Send(),
		)
	}

	billing_service.InvalidateEndpointWeightCache()

	return c.Status(fiber.StatusCreated).JSON(result)
}

// DeleteEndpointWeight removes a custom endpoint weight (admin only).
//
//	@Summary		Delete an endpoint weight
//	@Description	Removes a custom endpoint weight. The endpoint reverts to the default weight of 1. Requires billing.admin policy.
//	@Tags			Billing Endpoint Weight
//	@Accept			json
//	@Produce		json
//	@Param			id	query		string							true	"Endpoint weight ID"
//	@Success		204	{string}	string							"No content"
//	@Failure		400	{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500	{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/billing/endpoint-weight/ [delete]
func DeleteEndpointWeight(c *fiber.Ctx) error {
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

	if err := repository.DeleteByID[billing_entity.EndpointWeight](id.ID, database.DB); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete endpoint weight", err, "repository").Send(),
		)
	}

	billing_service.InvalidateEndpointWeightCache()

	return c.SendStatus(fiber.StatusNoContent)
}
