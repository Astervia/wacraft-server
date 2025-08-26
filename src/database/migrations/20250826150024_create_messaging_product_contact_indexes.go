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
	goose.AddMigrationContext(upCreateMessagingProductContactIndexes, downCreateMessagingProductContactIndexes)
}

func upCreateMessagingProductContactIndexes(ctx context.Context, tx *sql.Tx) error {
	// NOTE: CREATE INDEX CONCURRENTLY cannot run inside a transaction.
	// We execute using database.DB (separate connection) like your other migrations.
	db := database.DB

	stmts := []string{
		// Core: equality lookups by wa_id (scoped by messaging_product_id)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mpc_mp_waid_expr
		   ON messaging_product_contacts (messaging_product_id, ((product_details->>'wa_id')));`,

		// Optional but recommended if you sometimes match by phone_number too
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mpc_mp_phone_expr
		   ON messaging_product_contacts (messaging_product_id, ((product_details->>'phone_number')));`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("upCreateMessagingProductContactIndexes failed on:\n%s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("Expression indexes for messaging_product_contacts ensured (btree on JSON keys).")
	return nil
}

func downCreateMessagingProductContactIndexes(ctx context.Context, tx *sql.Tx) error {
	// Drop only the indexes created above.
	db := database.DB

	stmts := []string{
		`DROP INDEX CONCURRENTLY IF EXISTS idx_mpc_mp_phone_expr;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_mpc_mp_waid_expr;`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("downCreateMessagingProductContactIndexes failed on:\n%s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("Expression indexes for messaging_product_contacts dropped.")
	return nil
}
