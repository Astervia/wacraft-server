package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRemoveForeignKeyFkMessagesFrom, downRemoveForeignKeyFkMessagesFrom)
}

func upRemoveForeignKeyFkMessagesFrom(ctx context.Context, tx *sql.Tx) error {
	query := `
	DO $$
	BEGIN
		IF EXISTS (
			SELECT FROM pg_tables
			WHERE schemaname = 'public'
			AND tablename = 'messages'
		) THEN
			ALTER TABLE public.messages
			DROP CONSTRAINT IF EXISTS fk_messages_from;
		END IF;
	END $$;
	`

	_, err := tx.ExecContext(ctx, query)

	return err
}

func downRemoveForeignKeyFkMessagesFrom(ctx context.Context, tx *sql.Tx) error {
	query := `
	DO $$
	BEGIN
		IF EXISTS (
			SELECT FROM pg_tables
			WHERE schemaname = 'public'
			AND tablename = 'messages'
		) THEN
			ALTER TABLE public.messages
			ADD CONSTRAINT fk_messages_from
			FOREIGN KEY (from_id)
			REFERENCES public.messaging_product_contacts(id)
			ON DELETE RESTRICT
			ON UPDATE CASCADE;
		END IF;
	END $$;
	`

	_, err := tx.ExecContext(ctx, query)
	return err
}
