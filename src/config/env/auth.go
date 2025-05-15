package env

import (
	"os"

	"github.com/pterm/pterm"
)

var (
	JwtSecret string
	AuthToken string
)

func loadAuthEnv() {
	JwtSecret = os.Getenv("JWT_SECRET")
	AuthToken = os.Getenv("AUTH_TOKEN")

	pterm.DefaultLogger.Info("Auth environment done")
}
