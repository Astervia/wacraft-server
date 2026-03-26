package auth_middleware

import (
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/gofiber/fiber/v2"
	"net/http/httptest"
	"testing"
)

func TestSuMiddleware_NoLocals(t *testing.T) {
	app := fiber.New()
	app.Use(SuMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401")
	}
}

func TestSuMiddleware_NotSuperuser(t *testing.T) {
	usrRole := user_model.User
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user_entity.User{Role: &usrRole})
		return c.Next()
	})
	app.Use(SuMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403")
	}
}

func TestSuMiddleware_Success(t *testing.T) {
	admRole := user_model.Admin
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user_entity.User{Role: &admRole})
		return c.Next()
	})
	app.Use(SuMiddleware)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200")
	}
}
