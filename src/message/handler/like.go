package message_handler

import (
	"net/url"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// ContentLike returns messages where text matches sender, receiver, or product data fields.
//
//	@Summary		Search messages by content text
//	@Description	Matches the given text using ILIKE against `sender_data`, `receiver_data`, and `product_data` fields.
//	@Tags			Message
//	@Accept			json
//	@Produce		json
//	@Param			message		query		message_model.QueryPaginated	true	"Pagination and filter parameters"
//	@Param			likeText	path		string							true	"Text to apply like operator"
//	@Success		200			{array}		message_entity.Message			"List of matched messages"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid likeText or query"
//	@Failure		500			{object}	common_model.DescriptiveError	"Failed to query messages"
//	@Security		ApiKeyAuth
//	@Router			/message/content/like/{likeText} [get]
func ContentLike(c *fiber.Ctx) error {
	encodedText := c.Params("likeText")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode likeText", err, "net/url").Send(),
		)
	}

	query := new(message_model.QueryPaginated)
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

	messages, err := message_service.ContentLike(
		decodedText,
		message_entity.Message{
			MessageFields: message_model.MessageFields{
				FromID:             query.FromID,
				ToID:               query.ToID,
				MessagingProductID: query.MessagingProductID,
				AuditWithDeleted: common_model.AuditWithDeleted{
					Audit: common_model.Audit{
						ID: query.ID,
					},
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
			common_model.NewApiError("unable to get messages", err, "message_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}

// ContentKeyLike returns messages matching text in a specific field.
//
//	@Summary		Search messages by content field
//	@Description	Uses ILIKE to match the given text in the specified key field. The fields `from` and `to` are populated in the result.
//	@Tags			Message
//	@Accept			json
//	@Produce		json
//	@Param			message		query		message_model.QueryPaginated		true	"Pagination and filter parameters"
//	@Param			contentLike	path		message_model.ContentKeyLikeParams	true	"Params to query content like key"
//	@Success		200			{array}		message_entity.Message				"List of matched messages"
//	@Failure		400			{object}	common_model.DescriptiveError		"Invalid keyName, likeText, or query"
//	@Failure		500			{object}	common_model.DescriptiveError		"Failed to query messages"
//	@Security		ApiKeyAuth
//	@Router			/message/content/{keyName}/like/{likeText} [get]
func ContentKeyLike(c *fiber.Ctx) error {
	params := new(message_model.ContentKeyLikeParams)
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

	query := new(message_model.QueryPaginated)
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

	messages, err := message_service.ContentKeyLike(
		params.LikeText,
		params.KeyName,
		message_entity.Message{
			MessageFields: message_model.MessageFields{
				FromID:             query.FromID,
				ToID:               query.ToID,
				MessagingProductID: query.MessagingProductID,
				AuditWithDeleted: common_model.AuditWithDeleted{
					Audit: common_model.Audit{
						ID: query.ID,
					},
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
			common_model.NewApiError("unable to get messages", err, "message_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}
