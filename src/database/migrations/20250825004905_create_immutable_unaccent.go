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
	goose.AddMigrationContext(upCreateImmutableUnaccent, downCreateImmutableUnaccent)
}

func upCreateImmutableUnaccent(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// Ensure required extensions exist (idempotent)
		`CREATE EXTENSION IF NOT EXISTS unaccent;`,
		`CREATE EXTENSION IF NOT EXISTS pg_trgm;`,

		// Create an IMMUTABLE wrapper so we can use unaccent in index expressions
		// Using a constant regdictionary makes it immutable/deterministic.
		`CREATE OR REPLACE FUNCTION immutable_unaccent(text)
			RETURNS text
			LANGUAGE sql
			IMMUTABLE
			PARALLEL SAFE
			STRICT
		AS $$
			SELECT unaccent('unaccent'::regdictionary, $1)
		$$;`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration upCreateImmutableUnaccent failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("immutable_unaccent function created (extensions ensured).")
	return nil
}

func downCreateImmutableUnaccent(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// Drop only the function; keep extensions installed (harmless and possibly used elsewhere)
		`DROP FUNCTION IF EXISTS immutable_unaccent(text);`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("migration downCreateImmutableUnaccent failed on: %s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("immutable_unaccent function dropped.")
	return nil
}
