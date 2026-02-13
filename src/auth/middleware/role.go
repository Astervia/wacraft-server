package auth_middleware

import (
	"errors"
	"fmt"
	"slices"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/gofiber/fiber/v2"
)

func RoleMiddleware(roles ...user_model.Role) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(*user_entity.User)
		if !ok || user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(
				common_model.NewApiError("failed to retrieve user from context locals", errors.New("invalid convertion to type user_entity.User"), "middleware").Send(),
			)
		}
		if user.Role == nil {
			return c.Status(fiber.StatusForbidden).JSON(
				common_model.NewApiError("user role not allowed", errors.New("user has no role"), "middleware").Send(),
			)
		}
		allowed := slices.Contains(roles, *user.Role)
		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(
				common_model.NewApiError("user role not allowed", fmt.Errorf("user's role is not in %v", roles), "middleware").Send(),
			)
		}
		return c.Next()
	}
}
