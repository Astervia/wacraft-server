package auth_middleware

import (
	"errors"
	"strings"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
)

func TokenMiddleware(c *fiber.Ctx) error {
	if env.AuthToken == "" {
		return c.Next()
	}

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("Authorization header not provided", nil, "middleware").Send(),
		)
	}

	// Split the header to get the token
	splitToken := strings.Split(authHeader, "Bearer ")
	if len(splitToken) != 2 {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("unable to split token", errors.New("length of token splitted with Bearer is incorrect"), "middleware").Send(),
		)
	}
	tokenString := splitToken[1]

	if tokenString != env.AuthToken {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("invalid token", errors.New("invalid token"), "middleware").Send(),
		)
	}

	return c.Next()
}
