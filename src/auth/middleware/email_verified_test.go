package auth_middleware

import (
	"net/http/httptest"
	"testing"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
)

func TestEmailVerifiedMiddleware_NotRequired(t *testing.T) {
	env.RequireEmailVerification = false
	app := fiber.New()
	app.Use(EmailVerifiedMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200 ok")
	}
}

func TestEmailVerifiedMiddleware_NoUserLocals(t *testing.T) {
	env.RequireEmailVerification = true
	app := fiber.New()
	app.Use(EmailVerifiedMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 unauthorized")
	}
}

func TestEmailVerifiedMiddleware_Unverified(t *testing.T) {
	env.RequireEmailVerification = true
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user_entity.User{EmailVerified: false})
		return c.Next()
	})
	app.Use(EmailVerifiedMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403 forbidden")
	}
}

func TestEmailVerifiedMiddleware_Verified(t *testing.T) {
	env.RequireEmailVerification = true
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user_entity.User{EmailVerified: true})
		return c.Next()
	})
	app.Use(EmailVerifiedMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200 ok")
	}
}
