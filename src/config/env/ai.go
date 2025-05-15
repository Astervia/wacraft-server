package env

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
)

var AIURL string = "http://127.0.0.1:8000"

func loadAIEnv() {
	aiUrl := os.Getenv("AI_URL")
	if aiUrl != "" {
		AIURL = aiUrl
	}

	pterm.DefaultLogger.Info(
		fmt.Sprintf(
			"AI environment done with URL %s",
			AIURL),
	)
}
