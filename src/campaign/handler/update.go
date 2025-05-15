package campaign_handler

import (
	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// Update updates an existing campaign.
//	@Summary		Update campaign
//	@Description	Updates a campaign identified by its ID.
//	@Tags			Campaign
//	@Accept			json
//	@Produce		json
//	@Param			campaign	body		campaign_model.UpdateCampaign	true	"Updated campaign data"
//	@Success		200			{object}	campaign_entity.Campaign		"Updated campaign object"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign [patch]
func Update(c *fiber.Ctx) error {
	var updateData campaign_model.UpdateCampaign
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	campaign, err := repository.Updates(
		campaign_entity.Campaign{
			Name:               updateData.Name,
			MessagingProductId: updateData.MessagingProductId,
			Audit:              common_model.Audit{Id: updateData.Id},
		},
		&campaign_entity.Campaign{
			Audit: common_model.Audit{Id: updateData.Id},
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update campaign", err, "handler").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaign)
}
