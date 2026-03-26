package auth_middleware

import (
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
	"net/http/httptest"
	"testing"
)

func TestTokenMiddleware_NoAuthTokenEnv(t *testing.T) {
	env.AuthToken = ""
	app := fiber.New()
	app.Use(TokenMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %v", resp.StatusCode)
	}
}

func TestTokenMiddleware_MissingHeader(t *testing.T) {
	env.AuthToken = "secret_valid_token"
	app := fiber.New()
	app.Use(TokenMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401, got %v", resp.StatusCode)
	}
}

func TestTokenMiddleware_InvalidFormat(t *testing.T) {
	env.AuthToken = "secret_valid_token"
	app := fiber.New()
	app.Use(TokenMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "secret_valid_token") // missing Bearer
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401, got %v", resp.StatusCode)
	}
}

func TestTokenMiddleware_InvalidToken(t *testing.T) {
	env.AuthToken = "secret_valid_token"
	app := fiber.New()
	app.Use(TokenMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401, got %v", resp.StatusCode)
	}
}

func TestTokenMiddleware_ValidToken(t *testing.T) {
	env.AuthToken = "secret_valid_token"
	app := fiber.New()
	app.Use(TokenMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer secret_valid_token")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %v", resp.StatusCode)
	}
}
