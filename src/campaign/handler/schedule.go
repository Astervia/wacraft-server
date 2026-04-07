package campaign_handler

import (
	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// Schedule sets a campaign's scheduled_at time and transitions it to "scheduled" status.
//
//	@Summary		Schedule a campaign
//	@Description	Sets the scheduled_at time and transitions the campaign to "scheduled" status. The scheduler worker will automatically execute the campaign at that time. Requires campaign to be in "draft" or "failed" status.
//	@Tags			Campaign
//	@Accept			json
//	@Produce		json
//	@Param			body	body		campaign_model.ScheduleCampaign	true	"Schedule data"
//	@Success		200		{object}	campaign_entity.Campaign		"Updated campaign"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		404		{object}	common_model.DescriptiveError	"Campaign not found or access denied"
//	@Failure		409		{object}	common_model.DescriptiveError	"Campaign is already running or completed"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/campaign/schedule [post]
func Schedule(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var req campaign_model.ScheduleCampaign
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	// Load and validate campaign belongs to workspace.
	var campaign campaign_entity.Campaign
	if err := database.DB.
		Where("id = ? AND workspace_id = ?", req.ID, workspace.ID).
		First(&campaign).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("campaign not found or access denied", err, "handler").Send(),
		)
	}

	// Block scheduling if campaign is currently running or already completed.
	if campaign.Status == "running" || campaign.Status == "completed" {
		return c.Status(fiber.StatusConflict).JSON(
			common_model.NewApiError("campaign cannot be scheduled in its current status: "+campaign.Status, nil, "handler").Send(),
		)
	}

	// Update status and scheduled_at.
	if err := database.DB.Model(&campaign).
		Where("id = ? AND workspace_id = ?", req.ID, workspace.ID).
		Updates(map[string]interface{}{
			"status":       "scheduled",
			"scheduled_at": req.ScheduledAt,
		}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to schedule campaign", err, "repository").Send(),
		)
	}

	campaign.Status = "scheduled"
	campaign.ScheduledAt = req.ScheduledAt

	return c.Status(fiber.StatusOK).JSON(campaign)
}

// Unschedule cancels a pending schedule and resets the campaign to "draft" status.
//
//	@Summary		Unschedule a campaign
//	@Description	Cancels a pending schedule, clearing scheduled_at and resetting status to "draft". The campaign must not be currently running.
//	@Tags			Campaign
//	@Accept			json
//	@Produce		json
//	@Param			body	body		campaign_model.UnscheduleCampaign	true	"Unschedule data"
//	@Success		200		{object}	campaign_entity.Campaign			"Updated campaign"
//	@Failure		400		{object}	common_model.DescriptiveError		"Invalid request body"
//	@Failure		404		{object}	common_model.DescriptiveError		"Campaign not found or access denied"
//	@Failure		409		{object}	common_model.DescriptiveError		"Campaign is currently running"
//	@Failure		500		{object}	common_model.DescriptiveError		"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/campaign/schedule [delete]
func Unschedule(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var req campaign_model.UnscheduleCampaign
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	// Load and validate campaign belongs to workspace.
	var campaign campaign_entity.Campaign
	if err := database.DB.
		Where("id = ? AND workspace_id = ?", req.ID, workspace.ID).
		First(&campaign).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("campaign not found or access denied", err, "handler").Send(),
		)
	}

	// Cannot unschedule a campaign that is currently running.
	if campaign.Status == "running" {
		return c.Status(fiber.StatusConflict).JSON(
			common_model.NewApiError("campaign is currently running and cannot be unscheduled; use the WebSocket cancel message instead", nil, "handler").Send(),
		)
	}

	// Reset to draft.
	if err := database.DB.Model(&campaign).
		Where("id = ? AND workspace_id = ?", req.ID, workspace.ID).
		Updates(map[string]interface{}{
			"status":       "draft",
			"scheduled_at": nil,
		}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to unschedule campaign", err, "repository").Send(),
		)
	}

	campaign.Status = "draft"
	campaign.ScheduledAt = nil

	return c.Status(fiber.StatusOK).JSON(campaign)
}
