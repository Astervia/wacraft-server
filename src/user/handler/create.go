package user_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

//	@Summary		Creates a new user
//	@Description	Creates a new user account
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			user	body		user_model.Create				true	"User data"
//	@Success		201		{object}	user_entity.User				"Created user"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/user [post]
//	@Security		ApiKeyAuth
func CreateUser(c *fiber.Ctx) error {
	// Parse the request body
	var newUser user_model.Create
	if err := c.BodyParser(&newUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewParseJsonError(err).Send())
	}

	// Create the new user
	newEntity := user_entity.User{
		Name:     newUser.Name,
		Email:    newUser.Email,
		Password: newUser.Password,
		Role:     newUser.Role,
	}

	// Save the new user to the database
	if err := database.DB.Create(&newEntity).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create user", err, "gorm.io/gorm").Send(),
		)
	}

	// Return the created user (or just a success message)
	return c.Status(fiber.StatusCreated).JSON(newEntity)
}
