package message_handler

import (
	"net/url"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// Count returns the number of messages matching the given filters.
//
//	@Summary		Count messages
//	@Description	Returns the total number of messages matching the specified filters.
//	@Tags			Message
//	@Accept			json
//	@Produce		json
//	@Param			message	query		message_model.QueryPaginated	true	"Pagination and filter parameters"
//	@Success		200		{integer}	int								"Count of messages"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to count messages"
//	@Security		ApiKeyAuth
//	@Router			/message/count [get]
func Count(c *fiber.Ctx) error {
	query := new(message_model.Query)
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

	messages, err := repository.Count(
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
		&query.DateOrder,
		&query.DateWhereWithDeletedAt,
		"", database.DB.Model(&message_entity.Message{}),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to count messages", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}

// CountContentLike returns the number of messages matching a likeText in content fields.
//
//	@Summary		Count messages with content match
//	@Description	Returns the number of messages where the provided text matches content fields like sender_data, receiver_data, or product_data.
//	@Tags			Message
//	@Accept			json
//	@Produce		json
//	@Param			message		query		message_model.QueryPaginated	true	"Pagination and filter parameters"
//	@Param			likeText	path		string							true	"Substring to search in content fields"
//	@Success		200			{integer}	int								"Count of messages"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid likeText or query"
//	@Failure		500			{object}	common_model.DescriptiveError	"Failed to count messages"
//	@Security		ApiKeyAuth
//	@Router			/message/count/content/like/{likeText} [get]
func CountContentLike(c *fiber.Ctx) error {
	encodedText := c.Params("likeText")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode likeText", err, "net/url").Send(),
		)
	}

	query := new(message_model.Query)
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

	messages, err := message_service.CountContentLike(
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
		&query.DateOrder,
		&query.DateWhereWithDeletedAt,
		nil,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to count messages", err, "message_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}
