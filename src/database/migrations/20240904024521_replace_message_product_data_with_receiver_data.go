package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upReplaceMessageProductDataWithReceiverData, downReplaceMessageProductDataWithReceiverData)
}

func columnExists(ctx context.Context, tx *sql.Tx, tableName, columnName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_name = $1
			AND column_name = $2
		)
	`
	err := tx.QueryRowContext(ctx, query, tableName, columnName).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func upReplaceMessageProductDataWithReceiverData(ctx context.Context, tx *sql.Tx) error {
	// Check if the column product_data exists before renaming it
	exists, err := columnExists(ctx, tx, "messages", "product_data")
	if err != nil {
		return err
	}
	receiverColumn, err := columnExists(ctx, tx, "messages", "receiver_data")
	if err != nil {
		return err
	}
	if receiverColumn {
		return nil
	}
	if exists {
		_, err := tx.ExecContext(ctx, `
			ALTER TABLE messages
			RENAME COLUMN product_data TO receiver_data;
		`)
		if err != nil {
			return err
		}
	}
	return nil
}

func downReplaceMessageProductDataWithReceiverData(ctx context.Context, tx *sql.Tx) error {
	// Check if the column receiver_data exists before renaming it back
	exists, err := columnExists(ctx, tx, "messages", "receiver_data")
	if err != nil {
		return err
	}
	productColumn, err := columnExists(ctx, tx, "messages", "product_data")
	if err != nil {
		return err
	}
	if productColumn {
		return nil
	}
	if exists {
		_, err := tx.ExecContext(ctx, `
			ALTER TABLE messages
			RENAME COLUMN receiver_data TO product_data;
		`)
		if err != nil {
			return err
		}
	}
	return nil
}
