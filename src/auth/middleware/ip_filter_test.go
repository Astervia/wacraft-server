package auth_middleware

import (
	"net/http/httptest"
	"testing"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
)

func TestIPAllowlist_Empty(t *testing.T) {
	app := fiber.New()
	app.Use(IPAllowlistMiddleware([]string{}))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200")
	}
}

func TestIPAllowlist_Denied(t *testing.T) {
    app := fiber.New(fiber.Config{ProxyHeader: fiber.HeaderXForwardedFor})
	app.Use(IPAllowlistMiddleware([]string{"192.168.1.0/24"}))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(fiber.HeaderXForwardedFor, "10.0.0.5")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403 Forbidden")
	}
}

func TestIPAllowlist_Allowed(t *testing.T) {
	app := fiber.New(fiber.Config{ProxyHeader: fiber.HeaderXForwardedFor})
	app.Use(IPAllowlistMiddleware([]string{"192.168.1.0/24"}))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(fiber.HeaderXForwardedFor, "192.168.1.50")
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 permitted, got %d", resp.StatusCode)
	}
}

func TestIPDenylist_Denied(t *testing.T) {
	app := fiber.New(fiber.Config{ProxyHeader: fiber.HeaderXForwardedFor})
	app.Use(IPDenylistMiddleware([]string{"10.0.0.0/8"}))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(fiber.HeaderXForwardedFor, "10.0.0.5")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403 Forbidden")
	}
}

func TestIPDenylist_Allowed(t *testing.T) {
	app := fiber.New(fiber.Config{ProxyHeader: fiber.HeaderXForwardedFor})
	app.Use(IPDenylistMiddleware([]string{"10.0.0.0/8"}))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(fiber.HeaderXForwardedFor, "192.168.1.50")
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 permitted")
	}
}

func TestEnvInitializers(t *testing.T) {
    env.IPAllowlist = []string{"127.0.0.1/32"}
    env.IPDenylist = []string{"1.1.1.1/32"}
    allow := NewAllowlistMiddleware()
    deny := NewDenylistMiddleware()
    if allow == nil || deny == nil {
        t.Errorf("Expected valid handlers")
    }
}
