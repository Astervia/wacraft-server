package main

import (
	"fmt"
	"os"

	_ "github.com/Astervia/wacraft-server/src/config"
	"github.com/Astervia/wacraft-server/src/database"
	_ "github.com/Astervia/wacraft-server/src/database"
	_ "github.com/Astervia/wacraft-server/src/database/migrate"
	_ "github.com/Astervia/wacraft-server/src/server"
	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
)

// @title						wacraft Server API
// @version					0.1.0
// @description				Backend server for the wacraft project. Handles WhatsApp Cloud API operations, including message sending, receiving, and webhook handling.
// @contact.name				Astervia Dev Team
// @contact.url				https://github.com/Astervia
// @contact.email				wacraft@astervia.tech
// @license.name				MIT
// @license.url				https://opensource.org/licenses/MIT
// @BasePath					/
// @schemes					http https
// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						Authorization
// @securityDefinitions.apikey	WorkspaceAuth
// @in							header
// @name						X-Workspace-ID
// @description					Workspace ID (UUID) for multi-tenant access. Required for all workspace-scoped endpoints.
func main() {
	// Check for CLI commands
	if len(os.Args) > 1 {
		command := os.Args[1]

		switch command {
		case "migrate:down":
			runMigrationDown()
			return
		case "migrate:status":
			runMigrationStatus()
			return
		case "migrate:down-to":
			if len(os.Args) < 3 {
				pterm.DefaultLogger.Error("Usage: ./wacraft-server migrate:down-to <version>")
				os.Exit(1)
			}
			runMigrationDownTo(os.Args[2])
			return
		default:
			pterm.DefaultLogger.Error(fmt.Sprintf("Unknown command: %s", command))
			pterm.DefaultLogger.Info("Available commands: migrate:down, migrate:status, migrate:down-to <version>")
			os.Exit(1)
		}
	}

	// Default behavior: server starts via init() functions
}

func runMigrationDown() {
	pterm.DefaultLogger.Info("Rolling back last migration...")
	goose.SetDialect("postgres")

	db, err := database.DB.DB()
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to get database connection: %s", err))
		os.Exit(1)
	}

	if err := goose.Down(db, "src/database/migrations"); err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to roll back migration: %s", err))
		os.Exit(1)
	}

	pterm.DefaultLogger.Info("Migration rolled back successfully")
}

func runMigrationStatus() {
	pterm.DefaultLogger.Info("Checking migration status...")
	goose.SetDialect("postgres")

	db, err := database.DB.DB()
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to get database connection: %s", err))
		os.Exit(1)
	}

	if err := goose.Status(db, "src/database/migrations"); err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to check migration status: %s", err))
		os.Exit(1)
	}
}

func runMigrationDownTo(version string) {
	pterm.DefaultLogger.Info(fmt.Sprintf("Rolling back to migration version %s...", version))
	goose.SetDialect("postgres")

	db, err := database.DB.DB()
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to get database connection: %s", err))
		os.Exit(1)
	}

	versionInt := int64(0)
	if _, err := fmt.Sscanf(version, "%d", &versionInt); err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Invalid version format: %s", version))
		os.Exit(1)
	}

	if err := goose.DownTo(db, "src/database/migrations", versionInt); err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to roll back to version %s: %s", version, err))
		os.Exit(1)
	}

	pterm.DefaultLogger.Info(fmt.Sprintf("Successfully rolled back to migration version %s", version))
}
