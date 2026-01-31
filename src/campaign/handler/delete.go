package campaign_handler

import (
	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// Delete removes a campaign by its ID.
//
//	@Summary		Delete campaign
//	@Description	Deletes a campaign using the provided ID in the request body.
//	@Tags			Campaign
//	@Accept			json
//	@Produce		json
//	@Param			body	body		common_model.RequiredID			true	"Campaign ID to delete"
//	@Success		204		{string}	string							"Campaign deleted successfully"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/campaign [delete]
func Delete(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var body common_model.RequiredID
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

	// Delete with workspace scoping
	var campaign campaign_entity.Campaign
	if err := database.DB.Where("id = ? AND workspace_id = ?", body.ID, workspace.ID).Delete(&campaign).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete campaign", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// DeleteMessage removes a campaign message by its ID.
//
//	@Summary		Delete campaign message
//	@Description	Deletes a campaign message using the provided ID in the request body.
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			body	body		common_model.RequiredID			true	"Campaign message ID to delete"
//	@Success		204		{string}	string							"Campaign message deleted successfully"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/campaign/message [delete]
func DeleteMessage(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var body common_model.RequiredID
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

	// Delete with workspace scoping via campaign join
	var campaignMessage campaign_entity.CampaignMessage
	db := database.DB.
		Joins("JOIN campaigns ON campaign_messages.campaign_id = campaigns.id AND campaigns.workspace_id = ?", workspace.ID).
		Where("campaign_messages.id = ?", body.ID)

	if err := db.Delete(&campaignMessage).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete campaign message", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
