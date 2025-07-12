package message_handler

import (
	"net/url"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// GetWamId returns a paginated list of messages matching the given wamId.
//
//	@Summary		Search messages by wamId
//	@Description	Fetches a paginated list of messages where the wamId matches and filters are applied.
//	@Tags			WhatsApp message
//	@Accept			json
//	@Produce		json
//	@Param			message	query		message_model.QueryPaginated	true	"Pagination and query parameters"
//	@Param			wamId	path		string							true	"wamId value to search for"
//	@Success		200		{array}		message_entity.Message			"List of matching messages"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid wamId or query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to retrieve messages"
//	@Security		ApiKeyAuth
//	@Router			/message/whatsapp/wam-id/{wamId} [get]
func GetWamId(c *fiber.Ctx) error {
	encodedText := c.Params("wamId")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode wamId", err, "net/url").Send(),
		)
	}

	query := new(message_model.QueryPaginated)
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

	messages, err := message_service.GetWamId(
		decodedText,
		message_entity.Message{
			MessageFields: message_model.MessageFields{
				FromId:             query.FromId,
				ToId:               query.ToId,
				MessagingProductId: query.MessagingProductId,
				AuditWithDeleted: common_model.AuditWithDeleted{
					Audit: common_model.Audit{
						Id: query.Id,
					},
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
			common_model.NewApiError("unable to get messages", err, "message_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}
