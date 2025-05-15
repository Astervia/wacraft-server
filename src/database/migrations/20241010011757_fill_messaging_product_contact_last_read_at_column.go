package migrations

import (
	"context"
	"database/sql"
	"time"

	"github.com/Astervia/wacraft-server/src/database"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upFillMessagingProductContactLastReadAtColumn, downFillMessagingProductContactLastReadAtColumn)
}

func upFillMessagingProductContactLastReadAtColumn(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	// Update all existing rows in MessagingProductContact where LastReadAt is NULL
	err := db.Model(&messaging_product_entity.MessagingProductContact{}).
		Where("last_read_at IS NULL").
		Update("last_read_at", time.Now()).Error
	if err != nil {
		return err
	}

	return nil
}

func downFillMessagingProductContactLastReadAtColumn(ctx context.Context, tx *sql.Tx) error {
	db := database.DB

	// Reset LastReadAt to NULL for all rows
	err := db.Model(&messaging_product_entity.MessagingProductContact{}).
		Update("last_read_at", nil).Error
	if err != nil {
		return err
	}

	return nil
}
