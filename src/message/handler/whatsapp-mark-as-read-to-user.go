package message_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	_ "github.com/Rfluid/whatsapp-cloud-api/src/common/model"
	"github.com/gofiber/fiber/v2"
)

// MarkWhatsAppMessageAsReadToUser marks the last message in the conversation as read.
//	@Summary		Marks conversation as read to user
//	@Description	Marks the latest message in a conversation as read so that the recipient sees the double blue tick in WhatsApp.
//	@Tags			WhatsApp message
//	@Accept			json
//	@Produce		json
//	@Param			message	query		message_model.QueryPaginated	true	"Pagination and filter parameters"
//	@Success		200		{object}	common_model.SuccessResponse	"Success response"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to mark conversation as read"
//	@Security		ApiKeyAuth
//	@Router			/message/whatsapp/mark-as-read [post]
func MarkWhatsAppMessageAsReadToUser(c *fiber.Ctx) error {
	query := new(message_model.QueryPaginated)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err),
		)
	}

	query.Paginate.Limit = 1

	r, err := message_service.MarkWhatsAppMessageAsReadToUser(
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
		"",
		nil,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to mark conversation as read to user", err, "message_service").Send(),
		)
	}

	// Return the created entity
	return c.Status(fiber.StatusOK).JSON(r)
}
