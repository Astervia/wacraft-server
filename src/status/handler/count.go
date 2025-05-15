package status_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	status_model "github.com/Astervia/wacraft-core/src/status/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// Count returns the total number of statuses matching the provided filters.
//	@Summary		Counts statuses
//	@Description	Counts statuses based on the query parameters.
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

	statuses, err := repository.Count(
		status_entity.Status{
			StatusFields: status_model.StatusFields{
				MessageId: query.MessageId,
				Audit: common_model.Audit{
					Id: query.Id,
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
