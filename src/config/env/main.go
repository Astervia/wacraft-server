package env

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/pterm/pterm"
)

func init() {
	loadEnv()
	loadAuthEnv()
	loadDbEnv()
	loadServerEnv()
	loadWhatsAppEnv()
	loadEmailEnv()
	loadRegistrationEnv()
}

func loadEnv() {
	pterm.DefaultLogger.Info(
		"Loading environment variables...",
	)

	err := godotenv.Load(".env")
	if err != nil {
		pterm.DefaultLogger.Warn(
			fmt.Sprintf("Some error occurred loading the environment file at root directory: %s", err),
		)
		pterm.DefaultLogger.Warn(
			"Using environment variables from the system",
		)
	}
}
