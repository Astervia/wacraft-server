package auth_middleware

import (
	"strconv"
	"time"

	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func newRateLimiter(keyPrefix string, max int, window time.Duration, errorMsg string) fiber.Handler {
	if !env.RateLimitEnabled {
		return func(c *fiber.Ctx) error { return c.Next() }
	}
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		KeyGenerator: func(c *fiber.Ctx) string {
			return keyPrefix + ":" + c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", retryAfterSeconds(c, window))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   errorMsg,
				"message": "Please try again later",
			})
		},
	})
}

// retryAfterSeconds returns the number of seconds until the rate limit resets.
// It reads the X-RateLimit-Reset header set by the limiter and falls back to
// the full window duration if the header is absent or already expired.
func retryAfterSeconds(c *fiber.Ctx, window time.Duration) string {
	if reset := c.GetRespHeader("X-RateLimit-Reset"); reset != "" {
		if resetTime, err := strconv.ParseInt(reset, 10, 64); err == nil {
			if diff := resetTime - time.Now().Unix(); diff > 0 {
				return strconv.FormatInt(diff, 10)
			}
		}
	}
	return strconv.Itoa(int(window.Seconds()))
}

var RegistrationRateLimiter = newRateLimiter(
	"registration",
	env.RateLimitRegistration,
	env.RateLimitRegistrationWindow,
	"Too many registration attempts",
)

var LoginRateLimiter = newLoginRateLimiter()

func newLoginRateLimiter() fiber.Handler {
	if !env.RateLimitEnabled {
		return func(c *fiber.Ctx) error { return c.Next() }
	}
	window := env.RateLimitLoginWindow
	return limiter.New(limiter.Config{
		Max:        env.RateLimitLogin,
		Expiration: window,
		KeyGenerator: func(c *fiber.Ctx) string {
			var body struct {
				Email string `json:"email"`
			}
			if err := c.BodyParser(&body); err == nil && body.Email != "" {
				return "login:" + c.IP() + ":" + body.Email
			}
			return "login:" + c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", retryAfterSeconds(c, window))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Too many login attempts",
				"message": "Please try again later",
			})
		},
	})
}

var PasswordResetRateLimiter = newRateLimiter(
	"password-reset",
	env.RateLimitPasswordReset,
	env.RateLimitPasswordResetWindow,
	"Too many password reset attempts",
)

var EmailVerificationRateLimiter = newRateLimiter(
	"email-verification",
	env.RateLimitEmailVerification,
	env.RateLimitEmailVerificationWindow,
	"Too many verification attempts",
)

var ResetPasswordRateLimiter = newRateLimiter(
	"reset-password",
	env.RateLimitResetPassword,
	env.RateLimitResetPasswordWindow,
	"Too many password reset attempts",
)
