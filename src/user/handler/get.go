package user_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// Get retrieves a paginated list of users.
//
//	@Summary		Retrieve users (Admin only)
//	@Description	Returns a paginated list of users based on optional filters. Restricted to admin users only.
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			user	query		user_model.QueryPaginated		true	"Pagination and query parameters"
//	@Success		200		{array}		user_entity.User				"List of users"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		403		{object}	common_model.DescriptiveError	"Forbidden - Admin role required"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/user [get]
//	@Security		ApiKeyAuth
func Get(c *fiber.Ctx) error {
	query := new(user_model.QueryPaginated)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewParseJsonError(err).Send())
	}

	if err := validators.Validator().Struct(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewValidationError(err).Send())
	}

	users, err := repository.GetPaginated(
		user_entity.User{
			Name:  query.Name,
			Email: query.Email,
			Audit: common_model.Audit{ID: query.ID},
			Role:  query.Role,
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(common_model.NewApiError("unable to get paginated", err, "repository").Send())
	}

	return c.Status(fiber.StatusOK).JSON(users)
}
