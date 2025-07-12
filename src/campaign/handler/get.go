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

// Get returns a paginated list of campaigns.
//
//	@Summary		List campaigns (paginated)
//	@Description	Retrieves a paginated list of campaigns based on query parameters.
//	@Tags			Campaign
//	@Accept			json
//	@Produce		json
//	@Param			campaign	query		campaign_model.QueryPaginated	true	"Pagination and filtering options"
//	@Success		200			{array}		campaign_entity.Campaign		"List of campaigns"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid query"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign [get]
func Get(c *fiber.Ctx) error {
	query := new(campaign_model.QueryPaginated)
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

	campaigns, err := repository.GetPaginated(
		campaign_entity.Campaign{
			Name:               query.Name,
			MessagingProductId: query.MessagingProductId,
			Audit: common_model.Audit{
				Id: query.Id,
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get paginated", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaigns)
}

// GetMessages returns a paginated list of campaign messages.
//
//	@Summary		List campaign messages (paginated)
//	@Description	Retrieves a paginated list of messages associated with campaigns.
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryMessagesPaginated	true	"Pagination and filtering options"
//	@Success		200					{array}		campaign_entity.CampaignMessage			"List of campaign messages"
//	@Failure		400					{object}	common_model.DescriptiveError			"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError			"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/message [get]
func GetMessages(c *fiber.Ctx) error {
	query := new(campaign_model.QueryMessagesPaginated)
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

	db := database.DB.Model(&campaign_entity.CampaignMessage{}).Joins("Message")

	campaigns, err := repository.GetPaginated(
		campaign_entity.CampaignMessage{
			MessageId:  query.MessageId,
			CampaignId: query.CampaignId,
			Audit: common_model.Audit{
				Id: query.Id,
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"",
		db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get paginated", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaigns)
}

// GetUnsentMessages returns a paginated list of unsent campaign messages.
//
//	@Summary		List unsent campaign messages
//	@Description	Retrieves a paginated list of campaign messages that were not sent.
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryMessagesPaginated	true	"Pagination and filtering options"
//	@Success		200					{array}		campaign_entity.CampaignMessage			"List of unsent campaign messages"
//	@Failure		400					{object}	common_model.DescriptiveError			"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError			"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/message/unsent [get]
func GetUnsentMessages(c *fiber.Ctx) error {
	query := new(campaign_model.QueryMessagesPaginated)
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

	db := database.DB.Model(&campaign_entity.CampaignMessage{}).Joins("Message").Where("message_id IS NULL")

	campaigns, err := repository.GetPaginated(
		campaign_entity.CampaignMessage{
			MessageId:  query.MessageId,
			CampaignId: query.CampaignId,
			Audit: common_model.Audit{
				Id: query.Id,
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"",
		db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get paginated", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaigns)
}

// GetSentMessages returns a paginated list of sent campaign messages.
//
//	@Summary		List sent campaign messages
//	@Description	Retrieves a paginated list of campaign messages that have been sent.
//	@Tags			Campaign Message
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryMessagesPaginated	true	"Pagination and filtering options"
//	@Success		200					{array}		campaign_entity.CampaignMessage			"List of sent campaign messages"
//	@Failure		400					{object}	common_model.DescriptiveError			"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError			"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/message/sent [get]
func GetSentMessages(c *fiber.Ctx) error {
	query := new(campaign_model.QueryMessagesPaginated)
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

	db := database.DB.Model(&campaign_entity.CampaignMessage{}).Joins("Message").Where("message_id IS NOT NULL")

	campaigns, err := repository.GetPaginated(
		campaign_entity.CampaignMessage{
			MessageId:  query.MessageId,
			CampaignId: query.CampaignId,
			Audit: common_model.Audit{
				Id: query.Id,
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"",
		db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get paginated", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaigns)
}

// GetErrors returns a paginated list of campaign message send errors.
//
//	@Summary		List campaign message send errors
//	@Description	Retrieves a paginated list of send errors associated with campaign messages.
//	@Tags			Campaign Message Error
//	@Accept			json
//	@Produce		json
//	@Param			campaign_message	query		campaign_model.QueryErrorsPaginated			true	"Pagination and filtering options"
//	@Success		200					{array}		campaign_entity.CampaignMessageSendError	"List of campaign message send errors"
//	@Failure		400					{object}	common_model.DescriptiveError				"Invalid query"
//	@Failure		500					{object}	common_model.DescriptiveError				"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/error [get]
func GetErrors(c *fiber.Ctx) error {
	query := new(campaign_model.QueryErrorsPaginated)
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

	campaigns, err := repository.GetPaginated(
		campaign_entity.CampaignMessageSendError{
			CampaignMessageId: query.CampaignMessageId,
			CampaignMessage: &campaign_entity.CampaignMessage{
				CampaignId: query.CampaignId,
			},
			Audit: common_model.Audit{
				Id: query.Id,
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get paginated", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(campaigns)
}
