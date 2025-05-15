package user_handler

import (
	"net/url"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	user_service "github.com/Astervia/wacraft-server/src/user/service"
	"github.com/gofiber/fiber/v2"
)

// @Summary		Queries user key like text paginated
// @Description	Uses regex with the ~ operator to query text at the key.
// @Tags			User
// @Accept			json
// @Produce		json
// @Param			user		query		user_model.QueryPaginated		true	"Pagination and query parameters"
// @Param			likeText	path		string							true	"Text to apply like operator on the given key"
// @Param			keyName		path		string							true	"The key to apply like operator"
// @Success		200			{array}		user_entity.User				"List of users"
// @Failure		400			{object}	common_model.DescriptiveError	"Invalid path or query parameters"
// @Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
// @Router			/user/content/{keyName}/like/{likeText} [get]
// @Security		ApiKeyAuth
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

	query := new(user_model.QueryPaginated)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	messages, err := user_service.ContentKeyLike(
		decodedText,
		decodedKey,
		user_entity.User{
			Audit: common_model.Audit{Id: query.Id},
			Email: query.Email,
			Name:  query.Name,
			Role:  query.Role,
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
