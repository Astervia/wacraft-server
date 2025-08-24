package webhook_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_model "github.com/Astervia/wacraft-core/src/webhook/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// UpdateWebhook updates an existing webhook using the provided data.
//
//	@Summary		Update an existing webhook
//	@Description	Updates a webhook identified by its ID with new URL, authorization, event, and method settings.
//	@Tags			Webhook
//	@Accept			json
//	@Produce		json
//	@Param			webhook	body		webhook_model.UpdateWebhook		true	"Updated webhook data"
//	@Success		200		{object}	webhook_entity.Webhook			"Updated webhook"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/webhook [put]
//	@Security		ApiKeyAuth
func UpdateWebhook(c *fiber.Ctx) error {
	var editWebhook webhook_model.UpdateWebhook
	if err := c.BodyParser(&editWebhook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(editWebhook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	webhook, err := repository.Updates(
		webhook_entity.Webhook{
			Url:           editWebhook.Url,
			Authorization: editWebhook.Authorization,
			Event:         editWebhook.Event,
			HttpMethod:    editWebhook.HttpMethod,
			Timeout:       editWebhook.Timeout,
		},
		&webhook_entity.Webhook{
			Audit: common_model.Audit{
				ID: editWebhook.ID,
			},
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update webhook", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(webhook)
}
