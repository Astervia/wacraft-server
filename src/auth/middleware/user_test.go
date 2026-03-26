package auth_middleware

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
	"os"

	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	crypto_service "github.com/Astervia/wacraft-core/src/crypto/service"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	env.JwtSecret = "testing_secret_key"
	database.DB.AutoMigrate(&user_entity.User{})
	os.Exit(m.Run())
}

func createMockUser(t *testing.T) *user_entity.User {
	email := fmt.Sprintf("test-%s@example.com", uuid.New().String())
	hashedPassword, _ := crypto_service.HashPassword("secret")
	user := &user_entity.User{
		Email:    email,
		Password: hashedPassword,
	}
	user.ID = uuid.New()
	if err := database.DB.Exec("INSERT INTO users (id, email, password) VALUES (?, ?, ?)", user.ID, user.Email, user.Password).Error; err != nil {
		t.Fatalf("Failed to insert mock user: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})
	return user
}

func TestUserMiddleware_MissingHeader(t *testing.T) {
	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400")
	}
}

func TestUserMiddleware_InvalidFormat(t *testing.T) {
	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "invalid format")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400")
	}
}

func TestUserMiddleware_ParseTokenErr(t *testing.T) {
	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer bad.token.string")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401")
	}
}

func TestUserMiddleware_MissingSub(t *testing.T) {
	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	claims := jwt.MapClaims{"role": "user"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(env.JwtSecret))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401")
	}
}

func TestUserMiddleware_Success(t *testing.T) {
	user := createMockUser(t)

	claims := jwt.MapClaims{
		"sub": user.ID.String(),
		"iss": "wacraft-server",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStrRaw, _ := token.SignedString([]byte(env.JwtSecret))

	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStrRaw)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestUserMiddleware_DeletedUser(t *testing.T) {
	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	claims := jwt.MapClaims{
		"sub": uuid.New().String(),
		"iss": "wacraft-server",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStrRaw, _ := token.SignedString([]byte(env.JwtSecret))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStrRaw)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401")
	}
}
