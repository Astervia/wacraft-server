package auth_middleware

import (
	"time"

	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RegistrationRateLimiter limits registration attempts per IP
var RegistrationRateLimiter = limiter.New(limiter.Config{
	Max:        env.RateLimitRegistration,
	Expiration: 1 * time.Hour,
	KeyGenerator: func(c *fiber.Ctx) string {
		return "registration:" + c.IP()
	},
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":   "Too many registration attempts",
			"message": "Please try again later",
		})
	},
})

// LoginRateLimiter limits login attempts per IP + email
var LoginRateLimiter = limiter.New(limiter.Config{
	Max:        env.RateLimitLogin,
	Expiration: 15 * time.Minute,
	KeyGenerator: func(c *fiber.Ctx) string {
		// Parse body to get email for rate limiting
		var body struct {
			Email string `json:"email"`
		}
		if err := c.BodyParser(&body); err == nil && body.Email != "" {
			return "login:" + c.IP() + ":" + body.Email
		}
		return "login:" + c.IP()
	},
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":   "Too many login attempts",
			"message": "Please try again later",
		})
	},
})

// PasswordResetRateLimiter limits password reset requests per IP
var PasswordResetRateLimiter = limiter.New(limiter.Config{
	Max:        5,
	Expiration: 1 * time.Hour,
	KeyGenerator: func(c *fiber.Ctx) string {
		return "password-reset:" + c.IP()
	},
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":   "Too many password reset attempts",
			"message": "Please try again later",
		})
	},
})

// EmailVerificationRateLimiter limits verification email requests per IP
var EmailVerificationRateLimiter = limiter.New(limiter.Config{
	Max:        5,
	Expiration: 1 * time.Hour,
	KeyGenerator: func(c *fiber.Ctx) string {
		return "email-verification:" + c.IP()
	},
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":   "Too many verification attempts",
			"message": "Please try again later",
		})
	},
})
