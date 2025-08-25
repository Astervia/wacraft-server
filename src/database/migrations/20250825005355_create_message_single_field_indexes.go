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
	goose.AddMigrationContext(upCreateMessageSingleFieldIndexes, downCreateMessageSingleFieldIndexes)
}

func upCreateMessageSingleFieldIndexes(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// 0) Required extensions
		`CREATE EXTENSION IF NOT EXISTS pg_trgm;`,
		`CREATE EXTENSION IF NOT EXISTS unaccent;`,

		// 1) Immutable wrapper around unaccent() so it can be used in index expressions
		//    Qualify the dictionary name explicitly to keep it deterministic.
		// `CREATE OR REPLACE FUNCTION immutable_unaccent(text)
		// 	RETURNS text
		// 	LANGUAGE sql
		// 	IMMUTABLE
		// 	PARALLEL SAFE
		// 	STRICT
		// AS $$
		// 	SELECT unaccent('public.unaccent', $1)
		// $$;`,

		// 2) Per-column trigram GINs with immutable_unaccent + NULL-safe casts
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_sender_trgm_unaccent
		   ON messages USING GIN ((immutable_unaccent(COALESCE(sender_data::text, ''))) gin_trgm_ops);`,

		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_receiver_trgm_unaccent
		   ON messages USING GIN ((immutable_unaccent(COALESCE(receiver_data::text, ''))) gin_trgm_ops);`,

		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_product_trgm_unaccent
		   ON messages USING GIN ((immutable_unaccent(COALESCE(product_data::text, ''))) gin_trgm_ops);`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration upCreateMessageSingleFieldIndexes failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("create_message_single_field_indexes (immutable_unaccent): all indexes ensured.")
	return nil
}

func downCreateMessageSingleFieldIndexes(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// Drop indexes (keep extensions installed)
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_product_trgm_unaccent;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_receiver_trgm_unaccent;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_sender_trgm_unaccent;`,

		// Optional: drop the wrapper (safe if nothing else depends on it)
		`DROP FUNCTION IF EXISTS immutable_unaccent(text);`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration downCreateMessageSingleFieldIndexes failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("create_message_single_field_indexes (immutable_unaccent): all indexes dropped.")
	return nil
}
