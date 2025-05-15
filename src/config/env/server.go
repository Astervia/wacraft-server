package env

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
)

var (
	ServerHost string
	ServerPort string
	SuPassword string
	Env        string
)

func loadServerEnv() {
	ServerHost = os.Getenv("HOST")
	ServerPort = os.Getenv("PORT")
	SuPassword = os.Getenv("SU_PASSWORD")
	Env = os.Getenv("ENV")

	pterm.DefaultLogger.Info(
		fmt.Sprintf("Server environment done with port %s", ServerPort),
	)
}
