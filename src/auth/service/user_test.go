package auth_service

import (
	"testing"
	"time"

	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func TestGetFromToken_Success(t *testing.T) {
	password := "secret123"
	user := createTestUser(t, password)

	validToken, err := generateToken(user.ID.String(), 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	foundUser, err := GetFromToken(validToken)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if foundUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, foundUser.ID)
	}
	if foundUser.Email != user.Email {
		t.Errorf("Expected user Email %s, got %s", user.Email, foundUser.Email)
	}
}

func TestGetFromToken_InvalidClaims(t *testing.T) {
	claims := jwt.MapClaims{"sub": uuid.New().String()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Forcing a bad signature to break claims validation
	badTokenStr, _ := token.SignedString([]byte("wrong_secret_again"))

	_, err := GetFromToken(badTokenStr)
	if err == nil {
		t.Error("Expected error due to invalid token signature")
	}
}

func TestGetFromToken_MissingSub(t *testing.T) {
	claims := jwt.MapClaims{"exp": time.Now().Add(1 * time.Hour).Unix()} // lacks "sub"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(env.JwtSecret))

	_, err := GetFromToken(tokenStr)
	if err == nil || err.Error() != "token is missing subject claim" {
		t.Errorf("Expected 'token is missing subject claim' error, got: %v", err)
	}
}

func TestGetFromToken_UserNotFound(t *testing.T) {
	nonExistentID := uuid.New().String()
	validTokenSTR, _ := generateToken(nonExistentID, 1*time.Hour)

	_, err := GetFromToken(validTokenSTR)
	if err == nil {
		t.Fatal("Expected error because user does not exist in DB")
	}
}
