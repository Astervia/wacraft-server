package user_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// DeleteCurrentUser removes the user who made the request.
//
//	@Summary		Delete current user
//	@Description	Deletes the authenticated user from the database.
//	@Tags			User
//	@Success		204	{string}	string							"No content"
//	@Failure		500	{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/user/me [delete]
//	@Security		ApiKeyAuth
func DeleteCurrentUser(c *fiber.Ctx) error {
	user := c.Locals("user").(*user_entity.User)

	if err := repository.DeleteById[user_entity.User](user.Id, database.DB); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete user", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// DeleteUserByID removes a user by their ID. Only admins can call this.
//
//	@Summary		Delete user by ID
//	@Description	Deletes a user by ID. The special user su@sudo cannot be deleted.
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			body	body	common_model.RequiredId			true	"User ID to delete"
//	@Success		204		{string}	string							"No content"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		401		{object}	common_model.DescriptiveError	"Cannot delete su@sudo user"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/user [delete]
//	@Security		ApiKeyAuth
func DeleteUserByID(c *fiber.Ctx) error {
	var reqBody common_model.RequiredId
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	user, err := repository.First(
		user_entity.User{
			Audit: common_model.Audit{
				Id: reqBody.Id,
			},
		},
		0, nil, nil, "", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to find user", err, "repository").Send(),
		)
	}
	if user.Email == "su@sudo" {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("one cannot delete su@sudo user", err, "handler").Send(),
		)
	}

	err = repository.DeleteById[user_entity.User](reqBody.Id, database.DB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete user", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
