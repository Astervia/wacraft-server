package auth_websocket_middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	crypto_service "github.com/Astervia/wacraft-core/src/crypto/service"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
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
	if err := database.DB.Exec("INSERT INTO users (id, email, password, name, role) VALUES (?, ?, ?, '', 'user')", user.ID, user.Email, user.Password).Error; err != nil {
		t.Fatalf("Failed to insert mock user: %v", err)
	}
	t.Cleanup(func() {
		database.DB.Exec("DELETE FROM users WHERE id = ?", user.ID)
	})
	return user
}

func setupWSReq(url string) *http.Request {
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	return req
}

func TestWSUserMiddleware_NotWS(t *testing.T) {
	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil) // lacking headers entirely
	resp, _ := app.Test(req)
	// upgrading block causes 426
	if resp.StatusCode == 200 {
		t.Errorf("Expected upgrade required status")
	}
}

func TestWSUserMiddleware_MissingHeader(t *testing.T) {
	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := setupWSReq("/")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401")
	}
}

func TestWSUserMiddleware_QueryFallback(t *testing.T) {
	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	// Simulating broken token structure appended as query parameter.
	req := setupWSReq("/?Authorization=Bearer+invalidtoken")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestWSUserMiddleware_Success(t *testing.T) {
	user := createMockUser(t)
	claims := jwt.MapClaims{"sub": user.ID.String(), "exp": time.Now().Add(time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(env.JwtSecret))

	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := setupWSReq("/")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200")
	}
}

func TestWSUserMiddleware_MissingSub(t *testing.T) {
	claims := jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(env.JwtSecret))

	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := setupWSReq("/")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401")
	}
}

func TestWSUserMiddleware_UserNotFound(t *testing.T) {
	claims := jwt.MapClaims{"sub": uuid.New().String(), "exp": time.Now().Add(time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(env.JwtSecret))

	app := fiber.New()
	app.Use(UserMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := setupWSReq("/")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401")
	}
}
