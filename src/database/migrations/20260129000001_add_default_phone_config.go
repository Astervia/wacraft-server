package migrations

import (
	"context"
	"database/sql"
	"fmt"

	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
	"gorm.io/gorm"
)

func init() {
	goose.AddMigrationContext(upAddDefaultPhoneConfig, downAddDefaultPhoneConfig)
}

func upAddDefaultPhoneConfig(ctx context.Context, tx *sql.Tx) error {
	pterm.DefaultLogger.Info("Running default phone config migration...")

	// Check if all required env variables are provided
	wabaID := env.WabaID
	wabaAccessToken := env.WabaAccessToken
	wabaAccountID := env.WabaAccountID
	metaAppSecret := env.MetaAppSecret
	metaVerifyToken := env.MetaVerifyToken
	displayPhone := env.DisplayPhone

	// Use default display phone if not provided
	if displayPhone == "" {
		displayPhone = "80000085"
	}

	// If any required env variable is missing, skip migration
	if wabaID == "" || wabaAccessToken == "" || wabaAccountID == "" ||
		metaAppSecret == "" || metaVerifyToken == "" {
		pterm.DefaultLogger.Warn("Skipping default phone config creation - required env variables not provided")
		pterm.DefaultLogger.Info("Required: WABA_ID, WABA_ACCESS_TOKEN, WABA_ACCOUNT_ID, META_APP_SECRET, META_VERIFY_TOKEN")
		pterm.DefaultLogger.Info("Optional: DISPLAY_PHONE (defaults to 80000085)")
		return nil
	}

	// Find the default workspace
	var workspaceID string
	err := tx.QueryRowContext(ctx, `SELECT id FROM workspaces WHERE slug = 'default' LIMIT 1`).Scan(&workspaceID)
	if err == sql.ErrNoRows {
		pterm.DefaultLogger.Warn("Skipping default phone config creation - default workspace not found")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to find default workspace: %w", err)
	}

	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to parse workspace ID: %w", err)
	}

	// Check if a phone config already exists for this workspace and WABA ID
	var existingPhoneConfigID string
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM phone_configs WHERE workspace_id = $1 AND waba_id = $2 LIMIT 1`,
		workspaceID, wabaID,
	).Scan(&existingPhoneConfigID)

	if err == nil {
		pterm.DefaultLogger.Info(fmt.Sprintf("Phone config already exists for default workspace with WABA ID %s", wabaID))
		// Associate with messaging products if not already associated
		return associatePhoneConfigWithMessagingProducts(ctx, tx, existingPhoneConfigID, workspaceID)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing phone config: %w", err)
	}

	// Create the phone config
	phoneConfig := phone_config_entity.PhoneConfig{
		Name:               "Default Phone Config",
		WorkspaceID:        &workspaceUUID,
		WabaID:             wabaID,
		WabaAccountID:      wabaAccountID,
		DisplayPhone:       displayPhone,
		AccessToken:        wabaAccessToken,
		MetaAppSecret:      metaAppSecret,
		WebhookVerifyToken: metaVerifyToken,
		IsActive:           true,
	}

	if err := database.DB.Create(&phoneConfig).Error; err != nil {
		return fmt.Errorf("failed to create default phone config: %w", err)
	}

	pterm.DefaultLogger.Info(fmt.Sprintf("Created default phone config: %s", phoneConfig.ID))

	// Associate with messaging products
	return associatePhoneConfigWithMessagingProducts(ctx, tx, phoneConfig.ID.String(), workspaceID)
}

func associatePhoneConfigWithMessagingProducts(ctx context.Context, tx *sql.Tx, phoneConfigID, workspaceID string) error {
	// Update all messaging products for this workspace to use this phone config
	result, err := tx.ExecContext(ctx, `
		UPDATE messaging_products
		SET phone_config_id = $1
		WHERE workspace_id = $2 AND phone_config_id IS NULL
	`, phoneConfigID, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to associate phone config with messaging products: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected > 0 {
		pterm.DefaultLogger.Info(fmt.Sprintf("Associated phone config with %d messaging products", rowsAffected))
	}

	return nil
}

func downAddDefaultPhoneConfig(ctx context.Context, tx *sql.Tx) error {
	pterm.DefaultLogger.Info("Reverting default phone config migration...")

	// Find the default workspace
	var workspaceID string
	err := tx.QueryRowContext(ctx, `SELECT id FROM workspaces WHERE slug = 'default' LIMIT 1`).Scan(&workspaceID)
	if err == sql.ErrNoRows {
		pterm.DefaultLogger.Warn("Default workspace not found, nothing to revert")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to find default workspace: %w", err)
	}

	// Remove phone_config_id from messaging products in this workspace
	_, err = tx.ExecContext(ctx, `
		UPDATE messaging_products
		SET phone_config_id = NULL
		WHERE workspace_id = $1
	`, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to remove phone config associations: %w", err)
	}

	// Delete phone configs for the default workspace
	err = database.DB.Where("workspace_id = ? AND name = ?", workspaceID, "Default Phone Config").
		Delete(&phone_config_entity.PhoneConfig{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to delete default phone config: %w", err)
	}

	pterm.DefaultLogger.Info("Default phone config migration reverted")
	return nil
}
