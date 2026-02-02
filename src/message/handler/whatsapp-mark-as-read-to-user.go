package message_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	_ "github.com/Rfluid/whatsapp-cloud-api/src/common"
	"github.com/gofiber/fiber/v2"
)

// MarkWhatsAppMessageAsReadToUser marks the last message in the conversation as read.
//
//	@Summary		Mark last message as read
//	@Description	Marks the latest WhatsApp message in the conversation as read so the user sees the double blue check.
//	@Tags			WhatsApp message
//	@Accept			json
//	@Produce		json
//	@Param			message	query		message_model.QueryPaginated	true	"Pagination and filter parameters"
//	@Success		200		{object}	common.SuccessResponse	"Success response"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to mark conversation as read"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/message/whatsapp/mark-as-read [post]
func MarkWhatsAppMessageAsReadToUser(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	query := new(message_model.QueryPaginated)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err),
		)
	}

	if err := validators.Validator().Struct(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err),
		)
	}

	query.Paginate.Limit = 1

	entity := message_entity.Message{
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
	}

	// Try workspace-specific first, fall back to legacy
	r, err := message_service.MarkWhatsAppMessageAsReadToUserByWorkspace(
		entity,
		workspace.ID,
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

	return c.Status(fiber.StatusOK).JSON(r)
}
