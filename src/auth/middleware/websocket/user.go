package auth_websocket_middleware

import (
	"strings"

	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	auth_service "github.com/Astervia/wacraft-server/src/auth/service"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// @Summary		Authenticates websocket handshake
// @Description	All websocket routes with the "websocket" prefix are authenticated with this user middleware. messageChannel is the channel that the user wants to subscribe to.
// @Tags			Websocket
// @Accept			json
// @Produce		json
// @Success		200 "Authentication successful"
// @Router			/websocket/{messageChannel} [get]
// @Security		ApiKeyAuth
func UserMiddleware(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	// Get the authorization header
	authHeader := string(c.Request().Header.Peek("Authorization"))
	if authHeader == "" {
		authHeader = string(c.Query("Authorization"))
	}

	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Authorization header is missing"})
	}

	// Split the header to get the token
	splitToken := strings.Split(authHeader, "Bearer ")
	if len(splitToken) != 2 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid authorization header"})
	}
	tokenString := splitToken[1]

	// Parse the JWT token
	token, err := auth_service.ParseToken(tokenString)
	// Check if the token is valid
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": err})
	}

	if !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Token is not valid"})
	}

	// Add the user ID to the context
	claims := token.Claims.(jwt.MapClaims)
	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Unable to convert id to uuid"})
	}
	// Fetch user from database using the userID
	var user user_entity.User
	err = database.DB.First(&user, userID).Error
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Error fetching user from database"})
	}

	// Store the user in the context
	c.Locals("user", &user)

	// Continue to the next middleware or route handler
	return c.Next()
}
