package messaging_product_service

import (
	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	"github.com/Astervia/wacraft-server/src/database"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"gorm.io/gorm"
)

// Gets the messaging product contact or saves it if it doesn't exist.
func GetContactOrSave(
	mpContact messaging_product_entity.MessagingProductContact,
	contact contact_entity.Contact,
	db *gorm.DB,
) (messaging_product_entity.MessagingProductContact, error) {
	if db == nil {
		db = database.DB
	}

	// Search for the mp contact and return if it exists
	if err := db.Model(&mpContact).Where(&mpContact).Joins("Contact").First(&mpContact).Error; err == nil {
		return mpContact, err
	}

	// Create a contact to then create an mp contact
	if err := db.Model(&contact).Create(&contact).Error; err != nil {
		return mpContact, err
	}

	// Create the mp contact
	mpContact.ContactId = contact.Id
	if err := db.Model(&mpContact).Create(&mpContact).Error; err != nil {
		return mpContact, err
	}

	mpContact.Contact = &contact

	return mpContact, nil
}
