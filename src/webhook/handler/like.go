package webhook_handler

import (
	"net/url"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_model "github.com/Astervia/wacraft-core/src/webhook/model"
	"github.com/Astervia/wacraft-server/src/validators"
	webhook_service "github.com/Astervia/wacraft-server/src/webhook/service"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// ContentKeyLike returns webhooks filtered by a specific key using a partial text match.
//
//	@Summary		Query webhooks by key and partial value
//	@Description	Filters webhooks using the ILIKE operator on a specified field and partial value.
//	@Tags			Webhook
//	@Accept			json
//	@Produce		json
//	@Param			webhook		query		webhook_model.QueryPaginated		true	"Pagination and query parameters"
//	@Param			contentLike	path		webhook_model.ContentKeyLikeParams	true	"Params to query content like key"
//	@Success		200			{array}		webhook_entity.Webhook				"List of webhooks"
//	@Failure		400			{object}	common_model.DescriptiveError		"Invalid query or path parameters"
//	@Failure		500			{object}	common_model.DescriptiveError		"Internal server error"
//	@Router			/webhook/content/{keyName}/like/{likeText} [get]
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
func ContentKeyLike(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	params := new(webhook_model.ContentKeyLikeParams)
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
	likeText, err := url.QueryUnescape(string(params.LikeText))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode likeText", err, "net/url").Send(),
		)
	}

	query := new(webhook_model.QueryPaginated)
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

	webhooks, err := webhook_service.ContentKeyLike(
		likeText,
		params.KeyName,
		webhook_entity.Webhook{
			Audit:       common_model.Audit{ID: query.ID},
			Url:         query.Url,
			Event:       query.Event,
			HttpMethod:  query.HttpMethod,
			Timeout:     query.Timeout,
			WorkspaceID: &workspace.ID,
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

	return c.Status(fiber.StatusOK).JSON(webhooks)
}
