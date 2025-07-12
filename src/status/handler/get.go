package status_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	status_model "github.com/Astervia/wacraft-core/src/status/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// Get returns a paginated list of statuses.
//
//	@Summary		Retrieve statuses
//	@Description	Returns a paginated list of statuses based on optional filters.
//	@Tags			Status
//	@Accept			json
//	@Produce		json
//	@Param			status	query		status_model.QueryPaginated		true	"Pagination and query parameters"
//	@Success		200		{array}		status_entity.Status			"List of statuses"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to retrieve statuses"
//	@Security		ApiKeyAuth
//	@Router			/status [get]
func Get(c *fiber.Ctx) error {
	query := new(status_model.QueryPaginated)
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

	statuses, err := repository.GetPaginated(
		status_entity.Status{
			StatusFields: status_model.StatusFields{
				MessageId: query.MessageId,
				Audit: common_model.Audit{
					Id: query.Id,
				},
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhereWithDeletedAt,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get statuses", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(statuses)
}
