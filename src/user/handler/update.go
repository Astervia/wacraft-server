package user_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	crypto_service "github.com/Astervia/wacraft-core/src/crypto/service"
	"github.com/Astervia/wacraft-core/src/repository"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// UpdateCurrentUser updates the details of the authenticated user.
//
//	@Summary		Update current user
//	@Description	Updates the profile details of the authenticated user.
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			body	body		user_model.UpdateWithPassword	true	"User data to update"
//	@Success		200		{object}	user_entity.User				"User updated successfully"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/user/me [put]
//	@Security		ApiKeyAuth
func UpdateCurrentUser(c *fiber.Ctx) error {
	var editUser user_model.UpdateWithPassword
	if err := c.BodyParser(&editUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewParseJsonError(err).Send())
	}

	if err := validators.Validator().Struct(&editUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewValidationError(err).Send())
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

	updatedUser, err := repository.Updates(
		data,
		&user_entity.User{Audit: common_model.Audit{Id: user.Id}},
		database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update user", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(updatedUser)
}

// UpdateUserByID updates a user by their ID.
//
//	@Summary		Update user by ID
//	@Description	Updates user data by ID. Restricted to superusers. Cannot update su@sudo.
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

	if err := validators.Validator().Struct(&editUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewValidationError(err).Send())
	}

	user, err := repository.First(
		user_entity.User{Audit: common_model.Audit{Id: editUser.Id}},
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

	updatedUser, err := repository.Updates(
		data,
		&user_entity.User{Audit: common_model.Audit{Id: editUser.Id}},
		database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update user", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(updatedUser)
}
