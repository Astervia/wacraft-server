package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Astervia/wacraft-server/src/database"
	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
)

func init() {
	goose.AddMigrationContext(upWebhookIndexes, downWebhookIndexes)
}

func upWebhookIndexes(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// Partial index for efficient pending delivery lookup
		// Only indexes rows where status is 'pending' or 'attempted'
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_deliveries_pending
		   ON webhook_deliveries(status, next_attempt_at)
		   WHERE status IN ('pending', 'attempted');`,

		// Partial index for active webhooks
		// Only indexes rows where is_active is TRUE
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_webhooks_active
		   ON webhooks(is_active, workspace_id)
		   WHERE is_active = TRUE;`,

		// Index for looking up deliveries by webhook
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_deliveries_webhook_id
		   ON webhook_deliveries(webhook_id);`,

		// Index for event type filtering
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_deliveries_event_type
		   ON webhook_deliveries(event_type);`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration upWebhookIndexes failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("webhook_indexes: all indexes ensured.")
	return nil
}

func downWebhookIndexes(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		`DROP INDEX CONCURRENTLY IF EXISTS idx_deliveries_event_type;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_deliveries_webhook_id;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_webhooks_active;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_deliveries_pending;`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration downWebhookIndexes failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("webhook_indexes: all indexes dropped.")
	return nil
}
