package campaign_handler

import (
	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	campaign_service "github.com/Astervia/wacraft-server/src/campaign/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// ContentKeyLike searches campaigns using a "like" pattern on a specific field.
//
//	@Summary		Search campaigns with regex-like operator
//	@Description	Applies a ILIKE filter on the specified key field and returns paginated results.
//	@Tags			Campaign
//	@Accept			json
//	@Produce		json
//	@Param			campaign	query		campaign_model.QueryPaginated		true	"Pagination and filtering options"
//	@Param			contentLike	path		campaign_model.ContentKeyLikeParams	true	"Params to query content like key"
//	@Success		200			{array}		campaign_entity.Campaign			"List of matching campaigns"
//	@Failure		400			{object}	common_model.DescriptiveError		"Invalid input (e.g., decoding or query error)"
//	@Failure		500			{object}	common_model.DescriptiveError		"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/content/{keyName}/like/{likeText} [get]
func ContentKeyLike(c *fiber.Ctx) error {
	params := new(campaign_model.ContentKeyLikeParams)
	if err := c.ParamsParser(params); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}
	if err := validators.Validator().Struct(params); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

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

	messages, err := campaign_service.ContentKeyLike(
		params.LikeText,
		params.KeyName,
		campaign_entity.Campaign{
			Audit:              common_model.Audit{ID: query.ID},
			MessagingProductID: query.MessagingProductID,
			Name:               query.Name,
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		nil,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get campaigns", err, "campaign_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}
