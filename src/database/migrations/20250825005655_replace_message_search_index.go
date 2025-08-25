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
	goose.AddMigrationContext(upReplaceMessageSearchIndex, downReplaceMessageSearchIndex)
}

func upReplaceMessageSearchIndex(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// Ensure required extensions
		`CREATE EXTENSION IF NOT EXISTS pg_trgm;`,
		`CREATE EXTENSION IF NOT EXISTS unaccent;`,

		// Create immutable wrapper around unaccent (so it can be used in index expressions)
		// Using a constant regdictionary is immutable and avoids schema assumptions.
		// `CREATE OR REPLACE FUNCTION immutable_unaccent(text)
		// 	RETURNS text
		// 	LANGUAGE sql
		// 	IMMUTABLE
		// 	PARALLEL SAFE
		// 	STRICT
		// AS $$
		// 	SELECT unaccent('unaccent'::regdictionary, $1)
		// $$;`,

		// 1) Create the new index first (NULL-safe + immutable_unaccent)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_search_trgm_unaccent
		   ON messages USING GIN ((
		     immutable_unaccent(
		       COALESCE(sender_data::text, '') || ' ' ||
		       COALESCE(receiver_data::text, '') || ' ' ||
		       COALESCE(product_data::text, '')
		     )
		   ) gin_trgm_ops);`,

		// 2) Drop the old index afterwards (cleanup)
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_search_trgm;`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.Error.Printfln("migration upReplaceMessageSearchIndex failed on: %s\nerr: %v", s, err)
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("replace_message_search_index: new immutable_unaccent index ensured; old index dropped.")
	return nil
}

func downReplaceMessageSearchIndex(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// Drop the new index
		`DROP INDEX CONCURRENTLY IF EXISTS idx_messages_search_trgm_unaccent;`,

		// Recreate the previous (non-unaccent, NULL-unsafe) index to restore prior state
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_search_trgm
		   ON messages USING GIN ((
		     (sender_data::text || ' ' || receiver_data::text || ' ' || product_data::text)
		   ) gin_trgm_ops);`,

		// Optional: keep the wrapper function (it’s harmless and might be used elsewhere).
		// If you really want to remove it on rollback, uncomment:
		// `DROP FUNCTION IF EXISTS immutable_unaccent(text);`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(
				fmt.Sprintf("migration downReplaceMessageSearchIndex failed on: %s\nerr: %v", s, err),
			)
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("replace_message_search_index: new index dropped; old index restored.")
	return nil
}
