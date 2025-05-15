package database_migrate

import (
	"fmt"
	"os"

	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
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
		&contact_entity.Contact{},
		&messaging_product_entity.MessagingProduct{},
		&messaging_product_entity.MessagingProductContact{},
		&message_entity.Message{},
		&campaign_entity.Campaign{},
		&campaign_entity.CampaignMessage{},
		&campaign_entity.CampaignMessageSendError{},
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

// Executes goose migrations.
//
// DANGER: You cannot use before migrations when running a new instance of the application in a brand new database.
//
// Uncomment the logic to use this migrations.
func gooseBeforeAutomaticMigrations() {
	pterm.DefaultLogger.Info("Executing goose before automatic migrations...")
	// Configure Goose
	// goose.SetDialect("postgres") // Set the database dialect

	// Run the migrations
	// db, _ := database.DB.DB()
	// if err := goose.Up(db, "src/database/migrations-before"); err != nil {
	// 	pterm.DefaultLogger.Error(fmt.Sprintf("Unable to execute goose migrations before automatic: %s", err))
	// 	os.Exit(1)
	// }

	pterm.DefaultLogger.Info("Goose migrations before automatic executed")
}
