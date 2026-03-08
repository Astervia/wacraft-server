package env

import (
	"os"

	"github.com/pterm/pterm"
)

var (
	AllowRegistration        bool
	RequireEmailVerification bool
)

func loadRegistrationEnv() {
	AllowRegistration = os.Getenv("ALLOW_REGISTRATION") != "false"
	RequireEmailVerification = os.Getenv("REQUIRE_EMAIL_VERIFICATION") != "false"

	if AllowRegistration {
		pterm.DefaultLogger.Info("Open registration is ENABLED")
	} else {
		pterm.DefaultLogger.Info("Open registration is DISABLED")
	}

	if RequireEmailVerification {
		pterm.DefaultLogger.Info("Email verification is REQUIRED")
	}
}
