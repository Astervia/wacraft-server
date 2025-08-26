package messaging_product_service

import (
	"errors"

	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

// Gets the messaging product contact or saves it if it doesn't exist.
func GetContactOrSaveV0(
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
	mpContact.ContactID = contact.ID
	if err := db.Model(&mpContact).Create(&mpContact).Error; err != nil {
		return mpContact, err
	}

	mpContact.Contact = &contact

	return mpContact, nil
}

// Gets the messaging product contact or saves it if it doesn't exist.
func GetContactOrSave(
	mpContact messaging_product_entity.MessagingProductContact,
	contact contact_entity.Contact,
	db *gorm.DB,
) (messaging_product_entity.MessagingProductContact, error) {
	var out messaging_product_entity.MessagingProductContact

	if db == nil {
		db = database.DB
	}

	// --- 1) Build optimized find query ---
	q := db.Model(&messaging_product_entity.MessagingProductContact{}).
		Where("messaging_product_id = ?", mpContact.MessagingProductID).
		Joins("Contact")

	// Add JSON-key predicates using your helper (->> 'wa_id' / 'phone_number')
	if mpContact.ProductDetails != nil {
		// Ensure the inner struct is non-nil if you plan to set fields later
		// (not required just for querying).
		mpContact.ProductDetails.ParseIndividualFieldQueries(&q)
	}

	// Try to find existing MPC
	if err := q.First(&out).Error; err == nil {
		return out, nil // found
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return out, err // real DB error
	}

	// --- 2) Not found → create the Contact, then the MPC ---
	if err := db.Model(&contact).Create(&contact).Error; err != nil {
		return out, err
	}

	mpContact.ContactID = contact.ID
	// Make sure ProductDetails is preserved on insert
	// (it already is in mpContact, so just Create)
	if err := db.Model(&messaging_product_entity.MessagingProductContact{}).Create(&mpContact).Error; err != nil {
		return out, err
	}

	// Reload with preload for a fully populated return
	err := db.
		Model(&messaging_product_entity.MessagingProductContact{}).
		Where("id = ?", mpContact.ID).
		Preload("Contact").
		First(&out).Error

	return out, err
}
