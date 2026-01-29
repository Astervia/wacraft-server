package messaging_product_service

import (
	"errors"

	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/google/uuid"
)

func CreateContactForMessagingProduct(
	contact messaging_product_entity.MessagingProductContact,
	messagingProductName messaging_product_model.MessagingProductName,
	workspaceID uuid.UUID,
) (messaging_product_entity.MessagingProductContact, error) {
	tx := database.DB.Begin()
	if tx.Error != nil {
		return contact, errors.New("unable to start transaction")
	}

	mp := messaging_product_entity.MessagingProduct{Name: messagingProductName}

	if err := tx.Model(&mp).Where("name = ? AND workspace_id = ?", messagingProductName, workspaceID).First(&mp).Error; err != nil {
		tx.Rollback()
		return contact, errors.New("unable to find messaging product")
	}
	contact.MessagingProductID = mp.ID

	if err := tx.Create(&contact).Error; err != nil {
		tx.Rollback()
		return contact, errors.New("unable to save contact")
	}

	if err := tx.Commit().Error; err != nil {
		return contact, errors.New("failed to commit transaction")
	}

	return contact, nil
}
