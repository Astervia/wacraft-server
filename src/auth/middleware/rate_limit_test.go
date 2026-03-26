package auth_middleware

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"bytes"
	"testing"
	"time"

	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
)

func TestRetryAfterSeconds_Fallback(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		val := retryAfterSeconds(c, 10*time.Second)
		if val != "10" {
			t.Errorf("Expected fallback 10 seconds, got %s", val)
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/", nil)
	app.Test(req)
}

func TestRetryAfterSeconds_WithHeader(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		future := time.Now().Add(5 * time.Second).Unix()
		c.Response().Header.Set("X-RateLimit-Reset", strconv.FormatInt(future, 10))
		val := retryAfterSeconds(c, 10*time.Second)
		if val != "5" && val != "4" && val != "6" {
			t.Errorf("Expected ~5 seconds, got %s", val)
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/", nil)
	app.Test(req)
}

func TestRegistrationRateLimiter_LimitReached(t *testing.T) {
	env.RateLimitEnabled = true
	env.RateLimitRegistration = 1
	env.RateLimitRegistrationWindow = 1 * time.Minute

	// Force fresh handler because package 'var' resolves before env tweaks
	limiterHandler := newRateLimiter(
		"registration",
		env.RateLimitRegistration,
		env.RateLimitRegistrationWindow,
		"Too many registration attempts",
	)

	app := fiber.New()
	app.Use(limiterHandler)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	// First request - pass
	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Second request - threshold
	req2 := httptest.NewRequest("GET", "/", nil)
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("Expected 429 Too Many Requests, got %d", resp2.StatusCode)
	}
}

func TestLoginRateLimiter_LimitReached(t *testing.T) {
	env.RateLimitEnabled = true
	env.RateLimitLogin = 1
	env.RateLimitLoginWindow = 1 * time.Minute

	limiterHandler := newLoginRateLimiter()

	app := fiber.New()
	app.Use(limiterHandler)
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	payload, _ := json.Marshal(map[string]string{"email": "test@example.com"})

	// First request - pass
	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Second request within window - limited
	req2 := httptest.NewRequest("POST", "/", bytes.NewBuffer(payload))
	req2.Header.Set("Content-Type", "application/json")
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", resp2.StatusCode)
	}
}

func TestRateLimiters_Disabled(t *testing.T) {
	env.RateLimitEnabled = false
	
	app := fiber.New()
	app.Use(newLoginRateLimiter())
	app.Use(newRateLimiter("test", 1, 1*time.Minute, "Err"))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 since limiters are disabled")
	}

	// second should also pass
	req2 := httptest.NewRequest("GET", "/", nil)
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != 200 {
		t.Errorf("Expected 200 since limiters are disabled")
	}
}

func TestExportedHandlersExistency(t *testing.T) {
	if RegistrationRateLimiter == nil ||
		LoginRateLimiter == nil ||
		PasswordResetRateLimiter == nil ||
		EmailVerificationRateLimiter == nil ||
		ResetPasswordRateLimiter == nil {
		t.Error("Not all limiters were successfully exported via var assignation")
	}
}
