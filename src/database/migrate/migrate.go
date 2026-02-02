package database_migrate

import (
	"fmt"
	"os"

	// PREMIUM STARTS
	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	// PREMIUM ENDS
	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	"github.com/Astervia/wacraft-server/src/database"
	_ "github.com/Astervia/wacraft-server/src/database/migrations"
	_ "github.com/Astervia/wacraft-server/src/database/migrations-before"
	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
)

const gooseTableBefore = "goose_db_version_before"
const gooseTableMain = "goose_db_version"

func init() {
	splitGooseVersionTable()
	gooseBeforeAutomaticMigrations()
	automaticMigrations()
	gooseMigrations()
}

// Configures automatic migrations with ORM.
func automaticMigrations() {
	pterm.DefaultLogger.Info("Adding automatic migrations")
	err := database.DB.AutoMigrate(
		&user_entity.User{},
		&user_entity.EmailVerification{},
		&user_entity.PasswordResetToken{},
		&workspace_entity.Workspace{},
		&workspace_entity.WorkspaceMember{},
		&workspace_entity.WorkspaceMemberPolicy{},
		&workspace_entity.WorkspaceInvitation{},
		&phone_config_entity.PhoneConfig{},
		&contact_entity.Contact{},
		&messaging_product_entity.MessagingProduct{},
		&messaging_product_entity.MessagingProductContact{},
		&message_entity.Message{},
		// PREMIUM STARTS
		&campaign_entity.Campaign{},
		&campaign_entity.CampaignMessage{},
		&campaign_entity.CampaignMessageSendError{},
		// PREMIUM ENDS
		&webhook_entity.Webhook{},
		&webhook_entity.WebhookLog{},
		&status_entity.Status{},
	)
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Unable to add automatic migrations: %s", err))
		os.Exit(1)
	}
	pterm.DefaultLogger.Info("Automatic migrations done")
}

// Executes goose migrations.
func gooseMigrations() {
	pterm.DefaultLogger.Info("Executing goose migrations...")
	goose.SetDialect("postgres")
	goose.SetTableName(gooseTableMain)

	db, _ := database.DB.DB()
	if err := goose.Up(db, "src/database/migrations"); err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Unable to execute goose migrations: %s", err))
		os.Exit(1)
	}

	pterm.DefaultLogger.Info("Goose migrations executed")
}

// splitGooseVersionTable migrates existing databases that tracked both
// migrations-before and migrations in the same goose_db_version table.
// It moves migrations-before entries to a dedicated goose_db_version_before table.
func splitGooseVersionTable() {
	db, _ := database.DB.DB()

	// Check if the old shared table exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, gooseTableMain).Scan(&exists)
	if err != nil || !exists {
		return // Virgin DB or no goose table yet, nothing to migrate
	}

	// Check if the split was already done
	var beforeExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, gooseTableBefore).Scan(&beforeExists)
	if err != nil {
		return
	}
	if beforeExists {
		return // Already split
	}

	// Check if the shared table actually has migrations-before entries
	// (versions 20241011174817 and 20260128000001)
	var count int
	err = db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE version_id IN (20241011174817, 20260128000001)
	`, gooseTableMain)).Scan(&count)
	if err != nil || count == 0 {
		return // No migrations-before entries in the shared table
	}

	pterm.DefaultLogger.Info("Splitting goose version table for migrations-before...")

	// Create the before table with the same schema
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			version_id BIGINT NOT NULL,
			is_applied BOOLEAN NOT NULL,
			tstamp TIMESTAMP DEFAULT now()
		)
	`, gooseTableBefore))
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to create %s table: %s", gooseTableBefore, err))
		os.Exit(1)
	}

	// Insert the initial version 0 row that goose expects
	_, err = db.Exec(fmt.Sprintf(`
		INSERT INTO %s (version_id, is_applied) VALUES (0, true)
	`, gooseTableBefore))
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to insert initial row into %s: %s", gooseTableBefore, err))
		os.Exit(1)
	}

	// Move migrations-before entries
	_, err = db.Exec(fmt.Sprintf(`
		INSERT INTO %s (version_id, is_applied, tstamp)
		SELECT version_id, is_applied, tstamp FROM %s
		WHERE version_id IN (20241011174817, 20260128000001)
	`, gooseTableBefore, gooseTableMain))
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to copy entries to %s: %s", gooseTableBefore, err))
		os.Exit(1)
	}

	// Remove them from the main table
	_, err = db.Exec(fmt.Sprintf(`
		DELETE FROM %s WHERE version_id IN (20241011174817, 20260128000001)
	`, gooseTableMain))
	if err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Failed to remove entries from %s: %s", gooseTableMain, err))
		os.Exit(1)
	}

	pterm.DefaultLogger.Info("Goose version table split completed")
}

// Executes goose migrations before AutoMigrate.
// These migrations handle existing database upgrades (adding workspace_id columns, etc.)
// For virgin databases, they detect no existing tables and skip gracefully.
func gooseBeforeAutomaticMigrations() {
	pterm.DefaultLogger.Info("Executing goose before automatic migrations...")
	goose.SetDialect("postgres")
	goose.SetTableName(gooseTableBefore)

	db, _ := database.DB.DB()
	if err := goose.Up(db, "src/database/migrations-before"); err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Unable to execute goose migrations before automatic: %s", err))
		os.Exit(1)
	}

	pterm.DefaultLogger.Info("Goose migrations before automatic executed")
}
