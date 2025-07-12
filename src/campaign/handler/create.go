package campaign_handler

import (
	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// Create registers a new campaign.
//
//	@Summary		Create a new campaign
//	@Description	Creates a new campaign using the provided data and returns the created object.
//	@Tags			Campaign
//	@Accept			json
//	@Produce		json
//	@Param			campaign	body		campaign_model.CreateCampaign	true	"Campaign data"
//	@Success		201			{object}	campaign_entity.Campaign		"Created campaign"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign [post]
func Create(c *fiber.Ctx) error {
	var newCampaign campaign_model.CreateCampaign
	if err := c.BodyParser(&newCampaign); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&newCampaign); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	campaign, err := repository.Create(
		campaign_entity.Campaign{
			Name:               newCampaign.Name,
			MessagingProductId: newCampaign.MessagingProductId,
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create campaign", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(campaign)
}

// CreateMessage registers a new campaign message.
//
//	@Summary		Create a new campaign message
//	@Description	Creates a new campaign message using the provided data and returns the created object.
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	body		campaign_model.CreateCampaignMessage	true	"Campaign message data"
//	@Success		201					{object}	campaign_entity.CampaignMessage			"Created campaign message"
//	@Failure		400					{object}	common_model.DescriptiveError			"Invalid request body"
//	@Failure		500					{object}	common_model.DescriptiveError			"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/message [post]
func CreateMessage(c *fiber.Ctx) error {
	var newCampaign campaign_model.CreateCampaignMessage
	if err := c.BodyParser(&newCampaign); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&newCampaign); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	campaign, err := repository.Create(
		campaign_entity.CampaignMessage{
			CampaignId: newCampaign.CampaignId,
			SenderData: newCampaign.SenderData,
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create campaign message", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(campaign)
}
