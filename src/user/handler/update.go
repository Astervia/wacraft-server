package user_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	crypto_service "github.com/Astervia/wacraft-core/src/crypto/service"
	"github.com/Astervia/wacraft-core/src/repository"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

//	@Summary		Update current user
//	@Description	Updates the details of the user who made the request
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			body	body		user_model.UpdateWithPassword	true	"User data to update"
//	@Success		200		{object}	fiber.Map						"User updated successfully"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/user/me [put]
//	@Security		ApiKeyAuth
func UpdateCurrentUser(c *fiber.Ctx) error {
	var editUser user_model.UpdateWithPassword
	if err := c.BodyParser(&editUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewParseJsonError(err).Send())
	}

	user := c.Locals("user").(*user_entity.User)
	data := user_entity.User{
		Name:     editUser.Name,
		Email:    editUser.Email,
		Password: editUser.Password,
	}

	if data.Password != "" {
		hashedPassword, err := crypto_service.HashPassword(data.Password)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("unable to hash password", err, "crypto_service").Send(),
			)
		}
		data.Password = hashedPassword
	}

	// Update user using service function
	updatedUser, err := repository.Updates(
		data,
		&user_entity.User{
			Audit: common_model.Audit{
				Id: user.Id,
			},
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update user", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(updatedUser)
}

//	@Summary		Update user by ID
//	@Description	Updates a user's details by their ID (accessible by superuser)
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			body	body		user_model.UpdateWithId			true	"User data to update"
//	@Success		200		{object}	user_entity.User				"User updated successfully"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		401		{object}	common_model.DescriptiveError	"Unauthorized"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/user [put]
//	@Security		ApiKeyAuth
func UpdateUserByID(c *fiber.Ctx) error {
	var editUser user_model.UpdateWithId
	if err := c.BodyParser(&editUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewParseJsonError(err).Send())
	}

	user, err := repository.First(
		user_entity.User{
			Audit: common_model.Audit{
				Id: editUser.Id,
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
			common_model.NewApiError("one cannot update su@sudo user", err, "handler").Send(),
		)
	}

	data := user_entity.User{
		Name:  editUser.Name,
		Email: editUser.Email,
		Role:  editUser.Role,
	}

	// Update user using service function
	updatedUser, err := repository.Updates(
		data,
		&user_entity.User{
			Audit: common_model.Audit{
				Id: editUser.Id,
			},
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update user", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(updatedUser)
}
