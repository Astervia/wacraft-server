package webhook_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// DeleteWebhookById deletes a webhook based on its ID.
//
//	@Summary		Delete a webhook
//	@Description	Deletes a webhook using the provided unique ID.
//	@Tags			Webhook
//	@Accept			json
//	@Produce		json
//	@Param			body	body		common_model.RequiredId			true	"Webhook ID to delete"
//	@Success		204		"Webhook deleted successfully"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/webhook [delete]
//	@Security		ApiKeyAuth
func DeleteWebhookById(c *fiber.Ctx) error {
	var reqBody common_model.RequiredId
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	err := repository.DeleteById[webhook_entity.Webhook](reqBody.Id, database.DB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete webhook", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
