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
	goose.AddMigrationContext(upCreateMessageIndexes, downCreateMessageIndexes)
}

func upCreateMessageIndexes(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// 0) Needed for trigram index
		`CREATE EXTENSION IF NOT EXISTS pg_trgm;`,

		// 1) Hot path indexes for message feeds and lookups
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_from_created
		   ON messages (from_id, created_at DESC);`,

		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_to_created
		   ON messages (to_id, created_at DESC);`,

		// Since we often filter by messaging_product_id together
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_from_prod_created
		   ON messages (from_id, messaging_product_id, created_at DESC);`,

		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_to_prod_created
		   ON messages (to_id, messaging_product_id, created_at DESC);`,

		// 2) Functional indexes for the “party” key (COALESCE(from_id, to_id))
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_party
		   ON messages ((COALESCE(from_id, to_id)));`,

		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_party_created
		   ON messages ((COALESCE(from_id, to_id)), created_at DESC);`,

		// 3) Search indexes to avoid regex scans on jsonb casts
		// Trigram GIN on a concatenation of jsonb columns (free-text)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_search_trgm
		   ON messages USING GIN ( ( (sender_data::text || ' ' || receiver_data::text || ' ' || product_data::text) ) gin_trgm_ops );`,

		// JSON containment (exact key/value filters)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_product_gin
		   ON messages USING GIN (product_data jsonb_path_ops);`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration upCreateMessageIndexes failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("create_message_indexes: all indexes ensured.")
	return nil
}

func downCreateMessageIndexes(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// Drop in reverse-ish order; keep EXTENSION pg_trgm installed (safe to leave)
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_product_gin;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_search_trgm;`,

		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_party_created;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_party;`,

		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_to_prod_created;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_from_prod_created;`,

		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_to_created;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_from_created;`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration downCreateMessageIndexes failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("create_message_indexes: all indexes dropped.")
	return nil
}
