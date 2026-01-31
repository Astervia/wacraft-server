package auth_middleware

import (
	"errors"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
)

// EmailVerifiedMiddleware checks if the authenticated user has verified their email.
// Skips the check if RequireEmailVerification is disabled.
// Must be used after UserMiddleware.
func EmailVerifiedMiddleware(c *fiber.Ctx) error {
	if !env.RequireEmailVerification {
		return c.Next()
	}

	user, ok := c.Locals("user").(*user_entity.User)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("failed to retrieve user from context locals", errors.New("invalid conversion to type user_entity.User"), "middleware").Send(),
		)
	}

	if !user.EmailVerified {
		return c.Status(fiber.StatusForbidden).JSON(
			common_model.NewApiError("Email verification required", errors.New("user email is not verified"), "middleware").Send(),
		)
	}

	return c.Next()
}
