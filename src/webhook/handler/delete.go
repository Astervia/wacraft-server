package webhook_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// DeleteWebhookByID deletes a webhook based on its ID.
//
//	@Summary		Delete a webhook
//	@Description	Deletes a webhook using the provided unique ID.
//	@Tags			Webhook
//	@Accept			json
//	@Produce		json
//	@Param			body	body	common_model.RequiredID	true	"Webhook ID to delete"
//	@Success		204		"Webhook deleted successfully"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/webhook [delete]
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
func DeleteWebhookByID(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var reqBody common_model.RequiredID
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

	// Delete with workspace scoping
	var webhook webhook_entity.Webhook
	if err := database.DB.Where("id = ? AND workspace_id = ?", reqBody.ID, workspace.ID).Delete(&webhook).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete webhook", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
