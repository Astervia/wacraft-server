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

// GetWamID returns a paginated list of statuses with the given WhatsApp message ID (wamID).
//
//	@Summary		Retrieve statuses by wamID
//	@Description	Returns a paginated list of statuses filtered by WhatsApp message ID (wamID) and other query parameters.
//	@Tags			WhatsApp status
//	@Accept			json
//	@Produce		json
//	@Param			status	query		status_model.QueryPaginated		true	"Pagination and query parameters"
//	@Param			wamID	path		string							true	"Desired wamID"
//	@Success		200		{array}		status_entity.Status			"List of statuses"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid wamID or query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to fetch statuses"
//	@Security		ApiKeyAuth
//	@Router			/status/whatsapp/wam-id/{wamID} [get]
func GetWamID(c *fiber.Ctx) error {
	encodedText := c.Params("wamID")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode wamID", err, "net/url").Send(),
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

	statuses, err := status_service.GetWamID(
		decodedText,
		status_entity.Status{
			StatusFields: status_model.StatusFields{
				MessageID: query.MessageID,
				Audit: common_model.Audit{
					ID: query.ID,
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
