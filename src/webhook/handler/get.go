package webhook_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_model "github.com/Astervia/wacraft-core/src/webhook/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// GetWebhooks returns a paginated list of registered webhooks.
//
//	@Summary		Get webhooks paginated
//	@Description	Returns a paginated list of registered webhooks.
//	@Tags			Webhook
//	@Accept			json
//	@Produce		json
//	@Param			paginate	query		webhook_model.QueryPaginated	true	"Query parameters"
//	@Success		200			{array}		webhook_entity.Webhook			"List of webhooks"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/webhook [get]
//	@Security		ApiKeyAuth
func GetWebhooks(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

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

	webhooks, err := repository.GetPaginated(
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
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get webhooks", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(webhooks)
}

// GetWebhookLogs returns a paginated list of webhook execution logs.
//
//	@Summary		Get webhook logs paginated
//	@Description	Returns a paginated list of webhook execution logs.
//	@Tags			Webhook log
//	@Accept			json
//	@Produce		json
//	@Param			paginate	query		webhook_model.QueryLogsPaginated	true	"Query parameters"
//	@Success		200			{array}		webhook_entity.WebhookLog			"List of webhook logs"
//	@Failure		400			{object}	common_model.DescriptiveError		"Invalid query parameters"
//	@Failure		500			{object}	common_model.DescriptiveError		"Internal server error"
//	@Router			/webhook/log [get]
//	@Security		ApiKeyAuth
func GetWebhookLogs(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	query := new(webhook_model.QueryLogsPaginated)
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

	db := database.DB.Joins("JOIN webhooks ON webhook_logs.webhook_id = webhooks.id AND webhooks.workspace_id = ?", workspace.ID)

	webhooks, err := repository.GetPaginated(
		webhook_entity.WebhookLog{
			Audit:            common_model.Audit{ID: query.ID},
			WebhookID:        query.WebhookID,
			HttpResponseCode: query.HttpResponseCode,
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get webhooks", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(webhooks)
}
