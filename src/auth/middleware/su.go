package auth_middleware

import (
	"errors"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/gofiber/fiber/v2"
)

// SuMiddleware checks if the authenticated user is a superuser.
func SuMiddleware(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*user_entity.User)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("failed to retrieve user from context locals", errors.New("invalid convertion to type user_entity.User"), "middleware").Send(),
		)
	}

	if user.Role == nil || *user.Role != user_model.Admin {
		return c.Status(fiber.StatusForbidden).JSON(
			common_model.NewApiError("user is not a superuser", errors.New(`user's role does not match "admin"`), "middleware").Send(),
		)
	}

	return c.Next()
}
