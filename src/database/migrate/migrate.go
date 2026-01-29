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

func init() {
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
	// Configure Goose
	goose.SetDialect("postgres") // Set the database dialect

	// Run the migrations
	db, _ := database.DB.DB()
	if err := goose.Up(db, "src/database/migrations"); err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Unable to execute goose migrations: %s", err))
		os.Exit(1)
	}

	pterm.DefaultLogger.Info("Goose migrations executed")
}

// Executes goose migrations before AutoMigrate.
// These migrations handle existing database upgrades (adding workspace_id columns, etc.)
// For virgin databases, they detect no existing tables and skip gracefully.
func gooseBeforeAutomaticMigrations() {
	pterm.DefaultLogger.Info("Executing goose before automatic migrations...")
	goose.SetDialect("postgres")

	db, _ := database.DB.DB()
	if err := goose.Up(db, "src/database/migrations-before"); err != nil {
		pterm.DefaultLogger.Error(fmt.Sprintf("Unable to execute goose migrations before automatic: %s", err))
		os.Exit(1)
	}

	pterm.DefaultLogger.Info("Goose migrations before automatic executed")
}
