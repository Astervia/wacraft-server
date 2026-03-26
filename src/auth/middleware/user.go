package auth_middleware

import (
	"errors"
	"strings"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	auth_service "github.com/Astervia/wacraft-server/src/auth/service"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// Checks if user provided the correct token.
func UserMiddleware(c *fiber.Ctx) error {
	// Get the authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Authorization header not provided", nil, "middleware").Send(),
		)
	}

	// Split the header to get the token
	splitToken := strings.Split(authHeader, "Bearer ")
	if len(splitToken) != 2 {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to split token", errors.New("length of token splitted with Bearer is incorrect"), "middleware").Send(),
		)
	}
	tokenString := splitToken[1]

	// Parse the JWT token
	token, err := auth_service.ParseToken(tokenString)
	// Check if the token is valid
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("unable to parse token", err, "auth_service").Send(),
		)
	}

	if !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("token is invalid", nil, "middleware").Send(),
		)
	}

	// Add the user ID to the context
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("invalid claims type", errors.New("claims is not jwt.MapClaims"), "middleware").Send(),
		)
	}
	subClaim, ok := claims["sub"]
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("sub claim is missing", errors.New("token does not contain subject"), "middleware").Send(),
		)
	}
	subStr, ok := subClaim.(string)
	if !ok || subStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("sub claim is invalid", errors.New("subject is not a string"), "middleware").Send(),
		)
	}

	userID, err := uuid.Parse(subStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to parse user id", err, "github.com/google/uuid").Send(),
		)
	}
	// Fetch user from database using the userID
	var user user_entity.User
	err = database.DB.First(&user, userID).Error
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("unable to find user", err, "gorm.io/gorm").Send(),
		)
	}

	// Store the user in the context
	c.Locals("user", &user)

	// Continue to the next middleware or route handler
	return c.Next()
}
