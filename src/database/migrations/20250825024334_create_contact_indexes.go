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
	goose.AddMigrationContext(upCreateContactIndexes, downCreateContactIndexes)
}

func upCreateContactIndexes(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// 0) Required extensions
		`CREATE EXTENSION IF NOT EXISTS unaccent;`,
		`CREATE EXTENSION IF NOT EXISTS pg_trgm;`,

		// 1) Immutable wrapper so we can use unaccent in index expressions
		`CREATE OR REPLACE FUNCTION immutable_unaccent(text)
			RETURNS text
			LANGUAGE sql
			IMMUTABLE
			PARALLEL SAFE
			STRICT
		AS $$
			SELECT unaccent('unaccent'::regdictionary, $1)
		$$;`,

		// 2) messaging_product_contacts.product_details (jsonb as text)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mpc_product_details_trgm_unaccent
		   ON messaging_product_contacts
		   USING GIN ( (immutable_unaccent(COALESCE(product_details::text, ''))) gin_trgm_ops );`,

		// 3) contacts.email (text)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contacts_email_trgm_unaccent
		   ON contacts
		   USING GIN ( (immutable_unaccent(COALESCE(email, ''))) gin_trgm_ops );`,

		// 4) contacts.name (text)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contacts_name_trgm_unaccent
		   ON contacts
		   USING GIN ( (immutable_unaccent(COALESCE(name, ''))) gin_trgm_ops );`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("upCreateContactIndexes failed on:\n%s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("messaging_product_contact search indexes ensured (unaccent + trigram).")
	return nil
}

func downCreateContactIndexes(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	stmts := []string{
		// Drop indexes only; keep the function/extensions as they may be used elsewhere
		`DROP INDEX CONCURRENTLY IF EXISTS idx_contacts_name_trgm_unaccent;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_contacts_email_trgm_unaccent;`,
		`DROP INDEX CONCURRENTLY IF EXISTS idx_mpc_product_details_trgm_unaccent;`,

		// If you want to remove the wrapper (only if nothing else depends on it), uncomment:
		// `DROP FUNCTION IF EXISTS immutable_unaccent(text);`,
	}

	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			pterm.DefaultLogger.Error(fmt.Sprintf("downCreateContactIndexes failed on:\n%s\nerr: %v", s, err))
			return err
		}
		pterm.DefaultLogger.Info("Executed: " + s)
	}

	pterm.DefaultLogger.Info("messaging_product_contact search indexes dropped.")
	return nil
}
