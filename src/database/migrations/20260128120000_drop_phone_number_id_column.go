package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upDropPhoneNumberID, downDropPhoneNumberID)
}

func upDropPhoneNumberID(ctx context.Context, db *sql.DB) error {
	// Drop the old unique index on phone_number_id if it exists
	_, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_phone_configs_phone_number_id`)
	if err != nil {
		return err
	}

	// Drop the phone_number_id column if it exists
	// AutoMigrate will handle creating the new partial unique index on waba_id
	_, err = db.ExecContext(ctx, `ALTER TABLE phone_configs DROP COLUMN IF EXISTS phone_number_id`)
	if err != nil {
		return err
	}

	return nil
}

func downDropPhoneNumberID(ctx context.Context, db *sql.DB) error {
	// Re-add phone_number_id column (copy from waba_id)
	_, err := db.ExecContext(ctx, `
		ALTER TABLE phone_configs
		ADD COLUMN IF NOT EXISTS phone_number_id VARCHAR(255)
	`)
	if err != nil {
		return err
	}

	// Copy waba_id to phone_number_id
	_, err = db.ExecContext(ctx, `
		UPDATE phone_configs
		SET phone_number_id = waba_id
		WHERE phone_number_id IS NULL
	`)
	if err != nil {
		return err
	}

	// Make phone_number_id NOT NULL
	_, err = db.ExecContext(ctx, `
		ALTER TABLE phone_configs
		ALTER COLUMN phone_number_id SET NOT NULL
	`)
	if err != nil {
		return err
	}

	// Re-create the unique index on phone_number_id
	_, err = db.ExecContext(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_phone_configs_phone_number_id
		ON phone_configs (phone_number_id)
	`)
	if err != nil {
		return err
	}

	return nil
}
