package status_handler

import (
	"net/url"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	status_model "github.com/Astervia/wacraft-core/src/status/model"
	status_service "github.com/Astervia/wacraft-server/src/status/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// GetWamId returns a paginated list of statuses with the given WhatsApp message ID (wamId).
//
//	@Summary		Retrieve statuses by wamId
//	@Description	Returns a paginated list of statuses filtered by WhatsApp message ID (wamId) and other query parameters.
//	@Tags			WhatsApp status
//	@Accept			json
//	@Produce		json
//	@Param			status	query		status_model.QueryPaginated		true	"Pagination and query parameters"
//	@Param			wamId	path		string							true	"Desired wamId"
//	@Success		200		{array}		status_entity.Status			"List of statuses"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid wamId or query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to fetch statuses"
//	@Security		ApiKeyAuth
//	@Router			/status/whatsapp/wam-id/{wamId} [get]
func GetWamId(c *fiber.Ctx) error {
	encodedText := c.Params("wamId")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode wamId", err, "net/url").Send(),
		)
	}

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

	statuses, err := status_service.GetWamId(
		decodedText,
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
		nil,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get statuses", err, "status_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(statuses)
}
