package migrations

import (
	"context"
	"database/sql"

	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
	"gorm.io/gorm"
)

func init() {
	goose.AddMigrationContext(upAddWhatsappMessagingProduct, downAddWhatsappMessagingProduct)
}

func upAddWhatsappMessagingProduct(ctx context.Context, tx *sql.Tx) error {
	var existingMessagingProduct messaging_product_entity.MessagingProduct
	err := database.DB.Where("name = ?", messaging_product_model.WhatsApp).First(&existingMessagingProduct).Error

	if err == gorm.ErrRecordNotFound {
		whatsAppProduct := messaging_product_entity.MessagingProduct{
			Name: messaging_product_model.WhatsApp,
		}

		err = database.DB.Create(&whatsAppProduct).Error
		if err != nil {
			return err
		}
	} else if err != nil {
		// Error during the check, return the error
		return err
	} else {
		// User with the same email already exists, do nothing
		pterm.DefaultLogger.Warn("Messaging product with name 'WhatsApp' already exists.")
	}

	return nil
}

func downAddWhatsappMessagingProduct(ctx context.Context, tx *sql.Tx) error {
	err := database.DB.Delete(&messaging_product_entity.MessagingProduct{}, "name = ?", messaging_product_model.WhatsApp).Error
	if err != nil {
		return err
	}

	return nil
}
