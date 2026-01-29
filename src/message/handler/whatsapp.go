package message_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_model "github.com/Astervia/wacraft-core/src/webhook/model"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	"github.com/Astervia/wacraft-server/src/validators"
	webhook_service "github.com/Astervia/wacraft-server/src/webhook/service"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// SendMessage sends a new WhatsApp message and stores it if successful.
//
//	@Summary		Send WhatsApp message
//	@Description	Sends a WhatsApp message and stores it in the database if the operation is successful.
//	@Tags			WhatsApp message
//	@Accept			json
//	@Produce		json
//	@Param			message	body		message_model.SendWhatsAppMessage	true	"Message data"
//	@Success		201		{object}	message_entity.Message				"Message sent successfully"
//	@Failure		400		{object}	common_model.DescriptiveError		"Invalid message payload"
//	@Failure		500		{object}	common_model.DescriptiveError		"Failed to send or save the message"
//	@Security		ApiKeyAuth
//	@Router			/message/whatsapp [post]
func SendMessage(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var body message_model.SendWhatsAppMessage
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	propagateCallback := func(data message_entity.Message) {
		// Broadcast to workspace-scoped WebSocket clients
		go NewMessageWorkspaceManager.BroadcastToWorkspace(workspace.ID, data)
		go webhook_service.SendAllByQuery(
			webhook_entity.Webhook{
				Event:       webhook_model.SendWhatsAppMessage,
				WorkspaceID: &workspace.ID,
			},
			data,
		)
	}

	var entity message_entity.Message
	var err error

	// Use workspace-specific messaging if workspace has a phone config,
	// otherwise fall back to legacy global API
	entity, err = message_service.FindMessagingProductByWorkspaceAndSendMessage(
		body,
		workspace.ID,
		propagateCallback,
	)
	if err != nil {
		// Fall back to legacy method if workspace-specific fails
		entity, err = message_service.FindMessagingProductAndSendMessage(
			body,
			propagateCallback,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("unable to find messaging product and send message", err, "message_service").Send(),
			)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(entity)
}
