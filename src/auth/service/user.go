package auth_service

import (
	"errors"

	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/golang-jwt/jwt/v4"
)

// Retrieves the user associated with the provided JWT token.
func GetFromToken(tokenString string) (*user_entity.User, error) {
	token, err := ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed to parse claims from token")
	}

	userID := claims["sub"].(string)
	var user user_entity.User

	// Find user in the database based on userID
	err = database.DB.First(&user, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}
