package message_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_model "github.com/Astervia/wacraft-core/src/webhook/model"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	webhook_service "github.com/Astervia/wacraft-server/src/webhook/service"
	"github.com/gofiber/fiber/v2"
)

// SendMessage sends a new WhatsApp message and stores it if successful.
//	@Summary		Sends new message
//	@Description	Tries to send a WhatsApp message and stores it in the database if successful.
//	@Tags			WhatsApp message
//	@Accept			json
//	@Produce		json
//	@Param			message	body		message_model.SendWhatsAppMessage	true	"Message data"
//	@Success		201		{object}	message_entity.Message				"Message sent"
//	@Failure		400		{object}	common_model.DescriptiveError		"Invalid message payload"
//	@Failure		500		{object}	common_model.DescriptiveError		"Failed to send or save message"
//	@Security		ApiKeyAuth
//	@Router			/message/whatsapp [post]
func SendMessage(c *fiber.Ctx) error {
	// Parse the request body
	var body message_model.SendWhatsAppMessage
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	entity, err := message_service.FindMessagingProductAndSendMessage(
		body,
		func(data message_entity.Message) {
			go NewMessageChannel.BroadcastJsonMultithread(data)
			go webhook_service.SendAllByQuery(
				webhook_entity.Webhook{
					Event: webhook_model.SendWhatsAppMessage,
				},
				data,
			)
		},
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to find messaging product and send message", err, "message_service").Send(),
		)
	}

	// Return the created entity
	return c.Status(fiber.StatusCreated).JSON(entity)
}
