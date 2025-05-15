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
	loadAIEnv()
}

func loadEnv() {
	pterm.DefaultLogger.Info(
		"Loading production environment file...",
	)

	err := godotenv.Load(".env")
	if err != nil {
		pterm.DefaultLogger.Warn(
			"No .env at root directory",
		)
		pterm.DefaultLogger.Info(
			"Loading `.env.local`...",
		)
		err = godotenv.Load(".env.local")
		if err != nil {
			pterm.DefaultLogger.Warn(
				fmt.Sprintf("Some error occurred loading the local environment file: %s", err),
			)
			pterm.DefaultLogger.Warn(
				"Using environment variables from the system",
			)
		}
	}

	pterm.DefaultLogger.Info(
		"Environment file successfully set",
	)
}
