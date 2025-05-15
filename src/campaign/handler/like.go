package campaign_handler

import (
	"net/url"

	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	campaign_service "github.com/Astervia/wacraft-server/src/campaign/service"
	"github.com/gofiber/fiber/v2"
)

// ContentKeyLike searches campaigns using a "like" pattern on a specific field.
//	@Summary		Search campaigns with regex-like operator
//	@Description	Applies a case-insensitive regex-like (~) filter on the specified key field and returns paginated results.
//	@Tags			Campaign
//	@Accept			json
//	@Produce		json
//	@Param			campaign	query		campaign_model.QueryPaginated	true	"Pagination and filtering options"
//	@Param			keyName		path		string							true	"Field name to apply the like operator (e.g., 'name')"
//	@Param			likeText	path		string							true	"Value to search using the like operator"
//	@Success		200			{array}		campaign_entity.Campaign		"List of matching campaigns"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid input (e.g., decoding or query error)"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/campaign/content/{keyName}/like/{likeText} [get]
func ContentKeyLike(c *fiber.Ctx) error {
	encodedText := c.Params("likeText")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode likeText", err, "net/url").Send(),
		)
	}

	encodedKey := c.Params("keyName")
	decodedKey, err := url.QueryUnescape(encodedKey)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode keyName", err, "net/url").Send(),
		)
	}

	query := new(campaign_model.QueryPaginated)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	messages, err := campaign_service.ContentKeyLike(
		decodedText,
		decodedKey,
		campaign_entity.Campaign{
			Audit:              common_model.Audit{Id: query.Id},
			MessagingProductId: query.MessagingProductId,
			Name:               query.Name,
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		nil,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get webhooks", err, "webhook_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}
