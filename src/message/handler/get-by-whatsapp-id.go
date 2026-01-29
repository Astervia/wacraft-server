package message_handler

import (
	"net/url"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	"github.com/Astervia/wacraft-server/src/database"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// GetWamID returns a paginated list of messages matching the given wamID.
//
//	@Summary		Search messages by wamID
//	@Description	Fetches a paginated list of messages where the wamID matches and filters are applied.
//	@Tags			WhatsApp message
//	@Accept			json
//	@Produce		json
//	@Param			message	query		message_model.QueryPaginated	true	"Pagination and query parameters"
//	@Param			wamID	path		string							true	"wamID value to search for"
//	@Success		200		{array}		message_entity.Message			"List of matching messages"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid wamID or query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to retrieve messages"
//	@Security		ApiKeyAuth
//	@Router			/message/whatsapp/wam-id/{wamID} [get]
func GetWamID(c *fiber.Ctx) error {
	encodedText := c.Params("wamID")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode wamID", err, "net/url").Send(),
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

	workspace := workspace_middleware.GetWorkspace(c)
	db := database.DB.Joins("JOIN messaging_products ON messages.messaging_product_id = messaging_products.id AND messaging_products.workspace_id = ?", workspace.ID)

	messages, err := message_service.GetWamID(
		decodedText,
		message_entity.Message{
			MessageFields: message_model.MessageFields{
				FromID:             query.FromID,
				ToID:               query.ToID,
				MessagingProductID: query.MessagingProductID,
				AuditWithDeleted: common_model.AuditWithDeleted{
					Audit: common_model.Audit{
						ID: query.ID,
					},
				},
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhereWithDeletedAt,
		db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get messages", err, "message_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}
