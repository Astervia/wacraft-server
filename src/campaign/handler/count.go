package campaign_handler

import (
	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// CountMessages returns the total number of campaign messages matching the filters.
//
//	@Summary		Count campaign messages
//	@Description	Returns the total number of campaign messages that match the given filters.
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryMessages	true	"Filtering options"
//	@Success		200					{integer}	int								"Number of campaign messages"
//	@Failure		400					{object}	common_model.DescriptiveError	"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/campaign/message/count [get]
func CountMessages(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	query := new(campaign_model.QueryMessages)
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

	db := database.DB.Joins("JOIN campaigns ON campaign_messages.campaign_id = campaigns.id AND campaigns.workspace_id = ?", workspace.ID)

	campaigns, err := repository.Count(
		campaign_entity.CampaignMessage{
			MessageID:  query.MessageID,
			CampaignID: query.CampaignID,
			Audit: common_model.Audit{
				ID: query.ID,
			},
		},
		&query.DateOrder,
		&query.DateWhere,
		"", db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to count messages", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaigns)
}

// CountUnsentMessages returns the total number of unsent campaign messages.
//
//	@Summary		Count unsent campaign messages
//	@Description	Returns the number of campaign messages where the message ID is null (unsent messages).
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryMessages	true	"Filtering options"
//	@Success		200					{integer}	int								"Number of unsent campaign messages"
//	@Failure		400					{object}	common_model.DescriptiveError	"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/campaign/message/count/unsent [get]
func CountUnsentMessages(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	query := new(campaign_model.QueryMessages)
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

	db := database.DB.
		Joins("JOIN campaigns ON campaign_messages.campaign_id = campaigns.id AND campaigns.workspace_id = ?", workspace.ID).
		Where("message_id IS NULL")

	campaigns, err := repository.Count(
		campaign_entity.CampaignMessage{
			MessageID:  query.MessageID,
			CampaignID: query.CampaignID,
			Audit: common_model.Audit{
				ID: query.ID,
			},
		},
		&query.DateOrder,
		&query.DateWhere,
		"",
		db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to count messages", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaigns)
}

// CountSentMessages returns the total number of sent campaign messages.
//
//	@Summary		Count sent campaign messages
//	@Description	Returns the number of campaign messages where the message ID is not null (sent messages).
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryMessages	true	"Filtering options"
//	@Success		200					{integer}	int								"Number of sent campaign messages"
//	@Failure		400					{object}	common_model.DescriptiveError	"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/campaign/message/count/sent [get]
func CountSentMessages(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	query := new(campaign_model.QueryMessages)
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

	db := database.DB.
		Joins("JOIN campaigns ON campaign_messages.campaign_id = campaigns.id AND campaigns.workspace_id = ?", workspace.ID).
		Where("message_id IS NOT NULL")

	campaigns, err := repository.Count(
		campaign_entity.CampaignMessage{
			MessageID:  query.MessageID,
			CampaignID: query.CampaignID,
			Audit: common_model.Audit{
				ID: query.ID,
			},
		},
		&query.DateOrder,
		&query.DateWhere,
		"",
		db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to count messages", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaigns)
}
