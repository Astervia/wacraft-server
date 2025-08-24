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

// Count returns the total number of statuses matching the provided filters.
//
//	@Summary		Count statuses
//	@Description	Returns the number of status records that match the provided query parameters.
//	@Tags			Status
//	@Accept			json
//	@Produce		json
//	@Param			status	query		status_model.QueryPaginated		true	"Pagination and query parameters"
//	@Success		200		{integer}	int								"Count of statuses"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to count statuses"
//	@Security		ApiKeyAuth
//	@Router			/status/count [get]
func Count(c *fiber.Ctx) error {
	query := new(status_model.Query)
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

	statuses, err := repository.Count(
		status_entity.Status{
			StatusFields: status_model.StatusFields{
				MessageID: query.MessageID,
				Audit: common_model.Audit{
					ID: query.ID,
				},
			},
		},
		&query.DateOrder,
		&query.DateWhereWithDeletedAt,
		"",
		database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to count statuses", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(statuses)
}
