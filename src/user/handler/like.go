package user_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	user_service "github.com/Astervia/wacraft-server/src/user/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// ContentKeyLike searches users by a given key and a partial text pattern.
//
//	@Summary		Search users by key and text
//	@Description	Returns a paginated list of users where the specified key matches a partial value using ILIKE operator.
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			user		query		user_model.QueryPaginated		true	"Pagination and query parameters"
//	@Param			contentLike	path		user_model.ContentKeyLikeParams	true	"Params to query content like key"
//	@Success		200			{array}		user_entity.User				"List of users"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid path or query parameters"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/user/content/{keyName}/like/{likeText} [get]
//	@Security		ApiKeyAuth
func ContentKeyLike(c *fiber.Ctx) error {
	params := new(user_model.ContentKeyLikeParams)
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

	query := new(user_model.QueryPaginated)
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

	messages, err := user_service.ContentKeyLike(
		params.LikeText,
		params.KeyName,
		user_entity.User{
			Audit: common_model.Audit{ID: query.ID},
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
			common_model.NewApiError("unable to get users", err, "user_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}
