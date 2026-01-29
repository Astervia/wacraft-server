package env

import (
	"os"

	"github.com/pterm/pterm"
)

var (
	// SMTP Configuration
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// Application URL for email links
	AppBaseURL string
)

func loadEmailEnv() {
	SMTPHost = os.Getenv("SMTP_HOST")
	SMTPPort = os.Getenv("SMTP_PORT")
	if SMTPPort == "" {
		SMTPPort = "587"
	}
	SMTPUser = os.Getenv("SMTP_USER")
	SMTPPassword = os.Getenv("SMTP_PASSWORD")
	SMTPFrom = os.Getenv("SMTP_FROM")

	AppBaseURL = os.Getenv("APP_BASE_URL")
	if AppBaseURL == "" {
		AppBaseURL = "http://localhost:3000"
	}

	if SMTPHost == "" {
		pterm.DefaultLogger.Warn("SMTP not configured, emails will be logged only")
	} else {
		pterm.DefaultLogger.Info("Email service configured with SMTP host: " + SMTPHost)
	}
}
