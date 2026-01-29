package env

import (
	"os"
	"strconv"

	"github.com/pterm/pterm"
)

var (
	// Registration settings
	AllowRegistration         bool
	RequireEmailVerification  bool
	RateLimitRegistration     int // Per hour per IP
	RateLimitLogin            int // Per 15 min per IP+email
)

func loadRegistrationEnv() {
	AllowRegistration = os.Getenv("ALLOW_REGISTRATION") != "false"
	RequireEmailVerification = os.Getenv("REQUIRE_EMAIL_VERIFICATION") != "false"

	RateLimitRegistration = 5
	if val := os.Getenv("RATE_LIMIT_REGISTRATION"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			RateLimitRegistration = parsed
		}
	}

	RateLimitLogin = 10
	if val := os.Getenv("RATE_LIMIT_LOGIN"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			RateLimitLogin = parsed
		}
	}

	if AllowRegistration {
		pterm.DefaultLogger.Info("Open registration is ENABLED")
	} else {
		pterm.DefaultLogger.Info("Open registration is DISABLED")
	}

	if RequireEmailVerification {
		pterm.DefaultLogger.Info("Email verification is REQUIRED")
	}
}
