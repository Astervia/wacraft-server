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
	goose.AddMigrationContext(upCampaignSchedule, downCampaignSchedule)
}

func upCampaignSchedule(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// Add status column with default 'draft' for existing rows
		`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'draft';`,

		// Add scheduled_at nullable timestamp
		`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;`,

		// Partial index for efficient scheduled campaign lookup (only indexes scheduled rows)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_campaigns_scheduled
		   ON campaigns(scheduled_at, status) WHERE status = 'scheduled';`,

		// Index for status + workspace filtering
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_campaigns_status
		   ON campaigns(status, workspace_id);`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration upCampaignSchedule failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("campaign_schedule: all columns and indexes created.")
	return nil
}

func downCampaignSchedule(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		`DROP INDEX CONCURRENTLY IF EXISTS idx_campaigns_status;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_campaigns_scheduled;`,
		`ALTER TABLE campaigns DROP COLUMN IF EXISTS scheduled_at;`,
		`ALTER TABLE campaigns DROP COLUMN IF EXISTS status;`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration downCampaignSchedule failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("campaign_schedule: all columns and indexes dropped.")
	return nil
}
