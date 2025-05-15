package campaign_handler

import (
	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// CountMessages returns the total number of campaign messages matching the filters.
//	@Summary		Count campaign messages
//	@Description	Counts campaign messages based on the query parameters.
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryMessages	true	"Filtering options"
//	@Success		200					{integer}	int								"Count of campaign messages"
//	@Failure		400					{object}	common_model.DescriptiveError	"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/message/count [get]
func CountMessages(c *fiber.Ctx) error {
	query := new(campaign_model.QueryMessages)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	campaigns, err := repository.Count(
		campaign_entity.CampaignMessage{
			MessageId:  query.MessageId,
			CampaignId: query.CampaignId,
			Audit: common_model.Audit{
				Id: query.Id,
			},
		},
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to count messages", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaigns)
}

// CountUnsentMessages returns the total number of unsent campaign messages.
//	@Summary		Count unsent campaign messages
//	@Description	Counts campaign messages where message ID is null.
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryMessages	true	"Filtering options"
//	@Success		200					{integer}	int								"Count of unsent campaign messages"
//	@Failure		400					{object}	common_model.DescriptiveError	"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/message/count/unsent [get]
func CountUnsentMessages(c *fiber.Ctx) error {
	query := new(campaign_model.QueryMessages)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	db := database.DB
	db = db.Where("message_id IS NULL")

	campaigns, err := repository.Count(
		campaign_entity.CampaignMessage{
			MessageId:  query.MessageId,
			CampaignId: query.CampaignId,
			Audit: common_model.Audit{
				Id: query.Id,
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
//	@Summary		Count sent campaign messages
//	@Description	Counts campaign messages where message ID is not null.
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryMessages	true	"Filtering options"
//	@Success		200					{integer}	int								"Count of sent campaign messages"
//	@Failure		400					{object}	common_model.DescriptiveError	"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/message/count/sent [get]
func CountSentMessages(c *fiber.Ctx) error {
	query := new(campaign_model.QueryMessages)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	db := database.DB
	db = db.Where("message_id IS NOT NULL")

	campaigns, err := repository.Count(
		campaign_entity.CampaignMessage{
			MessageId:  query.MessageId,
			CampaignId: query.CampaignId,
			Audit: common_model.Audit{
				Id: query.Id,
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
