package auth_service

import (
	"fmt"
	"os"
	"testing"
	"time"

	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	crypto_service "github.com/Astervia/wacraft-core/src/crypto/service"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	// Lock standard test environment secret
	env.JwtSecret = "testing_secret_key"
	// Manually bootstrap the required schema for tests, bypassing problematic Goose migrations
	database.DB.AutoMigrate(&user_entity.User{})
	os.Exit(m.Run())
}

// Shared helper to bootstrap mock users with transient emails per test
func createTestUser(t *testing.T, plainPassword string) *user_entity.User {
	email := fmt.Sprintf("test-%s@example.com", uuid.New().String())
	hashedPassword, err := crypto_service.HashPassword(plainPassword)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := &user_entity.User{
		Email:    email,
		Password: hashedPassword,
	}
	user.ID = uuid.New()

	// Insert using Raw SQL to bypass any BeforeCreate/BeforeSave hooks that might double-hash the password
	if err := database.DB.Exec("INSERT INTO users (id, email, password, name, role) VALUES (?, ?, ?, '', 'user')", user.ID, user.Email, user.Password).Error; err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	// Purge DB context explicitly matching isolation rule
	t.Cleanup(func() {
		database.DB.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})

	return user
}

func TestLogin_Success(t *testing.T) {
	password := "correct_horse_battery_staple"
	user := createTestUser(t, password)

	resp, err := Login(user.Email, password)
	if err != nil {
		t.Fatalf("Expected successful login, got error: %v", err)
	}
	if resp == nil {
		t.Fatalf("Expected non-nil TokenResponse")
	}

	if resp.TokenType != "bearer" {
		t.Errorf("Expected token_type to be 'bearer', got '%s'", resp.TokenType)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("Expected expires_in 3600, got %d", resp.ExpiresIn)
	}
	if resp.AccessToken == "" {
		t.Error("Expected non-empty access_token")
	}
	if resp.RefreshToken == "" {
		t.Error("Expected non-empty refresh_token")
	}
}

func TestLogin_EmailNotFound(t *testing.T) {
	email := fmt.Sprintf("nonexistent-%s@example.com", uuid.New().String())
	resp, err := Login(email, "password123")
	if err == nil {
		t.Fatal("Expected error for nonexistent email, got nil")
	}
	if resp != nil {
		t.Fatal("Expected nil response on error")
	}
}

func TestLogin_IncorrectPassword(t *testing.T) {
	password := "mypassword"
	user := createTestUser(t, password)

	resp, err := Login(user.Email, "wrongpassword")
	if err == nil || err.Error() != "incorrect password" {
		t.Fatalf("Expected 'incorrect password' error, got %v", err)
	}
	if resp != nil {
		t.Fatal("Expected nil response on error")
	}
}

func TestRefreshToken_Success(t *testing.T) {
	userID := uuid.New().String()
	validToken, err := generateToken(userID, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	resp, err := RefreshToken(validToken)
	if err != nil {
		t.Fatalf("Expected successful refresh, got error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("Expected new access token")
	}
	if resp.RefreshToken != validToken {
		t.Error("Expected refresh_token to be identical to the provided input")
	}
	if resp.TokenType != "bearer" {
		t.Errorf("Expected token_type 'bearer'")
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("Expected expires_in 3600")
	}
}

func TestRefreshToken_InvalidSecret(t *testing.T) {
	claims := jwt.MapClaims{"sub": uuid.New().String()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	badTokenStr, _ := token.SignedString([]byte("wrong_secret"))

	_, err := RefreshToken(badTokenStr)
	if err == nil || err.Error() != "signature is invalid" {
		t.Errorf("Expected signature is invalid error, got: %v", err)
	}
}

func TestRefreshToken_UnexpectedAlg(t *testing.T) {
	claims := jwt.MapClaims{"sub": uuid.New().String()}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	badTokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := RefreshToken(badTokenStr)
	if err == nil {
		t.Error("Expected error due to unexpected signing method algorithm")
	}
}

func TestRefreshToken_MissingSub(t *testing.T) {
	claims := jwt.MapClaims{"iat": time.Now().Unix()} // lacks "sub"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(env.JwtSecret))

	_, err := RefreshToken(tokenStr)
	if err == nil || err.Error() != "token is missing subject claim" {
		t.Errorf("Expected 'token is missing subject claim' error, got: %v", err)
	}
}

func TestGenerateToken_ContainsClaims(t *testing.T) {
	userID := uuid.New().String()
	tokenStr, err := generateToken(userID, 1*time.Hour)
	if err != nil {
		t.Fatalf("Generate token failed: %v", err)
	}

	parsed, err := ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("Parse token failed: %v", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Failed to parse claims map")
	}
	if claims["sub"] != userID {
		t.Errorf("Expected sub %s, got %v", userID, claims["sub"])
	}
	if claims["iss"] != "wacraft-server" {
		t.Errorf("Expected iss 'wacraft-server', got %v", claims["iss"])
	}
}
