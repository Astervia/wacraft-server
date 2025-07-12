package status_handler

import (
	"net/url"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	status_model "github.com/Astervia/wacraft-core/src/status/model"
	status_service "github.com/Astervia/wacraft-server/src/status/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// ContentLike returns statuses that match a partial text pattern in key fields.
//
//	@Summary		Search statuses by content
//	@Description	Returns a paginated list of statuses matching a partial text (using regex ~) in sender_data, receiver_data, or product_data.
//	@Tags			Status
//	@Accept			json
//	@Produce		json
//	@Param			status		query		status_model.QueryPaginated		true	"Pagination and query parameters"
//	@Param			likeText	path		string							true	"Text to match in content fields"
//	@Success		200			{array}		status_entity.Status			"List of statuses"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid query or path parameter"
//	@Failure		500			{object}	common_model.DescriptiveError	"Failed to retrieve statuses"
//	@Security		ApiKeyAuth
//	@Router			/status/content/like/{likeText} [get]
func ContentLike(c *fiber.Ctx) error {
	encodedText := c.Params("likeText")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode likeText", err, "net/url").Send(),
		)
	}

	query := new(status_model.QueryPaginated)
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

	statuses, err := status_service.ContentLike(
		decodedText,
		status_entity.Status{
			StatusFields: status_model.StatusFields{
				MessageId: query.MessageId,
				Audit: common_model.Audit{
					Id: query.Id,
				},
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhereWithDeletedAt,
		nil,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get statuses", err, "status_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(statuses)
}

// ContentKeyLike returns statuses filtered by a key and a partial text pattern.
//
//	@Summary		Search statuses by key and content
//	@Description	Returns a paginated list of statuses where a given key matches a partial value using regex (~).
//	@Tags			Status
//	@Accept			json
//	@Produce		json
//	@Param			status		query		status_model.QueryPaginated		true	"Pagination and query parameters"
//	@Param			keyName		path		string							true	"Field name to search"
//	@Param			likeText	path		string							true	"Text to match on the key"
//	@Success		200			{array}		status_entity.Status			"List of statuses"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid query or path parameter"
//	@Failure		500			{object}	common_model.DescriptiveError	"Failed to retrieve statuses"
//	@Security		ApiKeyAuth
//	@Router			/status/content/{keyName}/like/{likeText} [get]
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

	query := new(status_model.QueryPaginated)
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

	statuses, err := status_service.ContentKeyLike(
		decodedText,
		decodedKey,
		status_entity.Status{
			StatusFields: status_model.StatusFields{
				MessageId: query.MessageId,
				Audit: common_model.Audit{
					Id: query.Id,
				},
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhereWithDeletedAt,
		nil,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get statuses", err, "status_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(statuses)
}
