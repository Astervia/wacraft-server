package messaging_product_service

import (
	"github.com/Astervia/wacraft-server/src/database"
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"gorm.io/gorm"
)

func ContactContentLike(
	likeText string,
	entity messaging_product_entity.MessagingProductContact,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]messaging_product_entity.MessagingProductContact, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	db = db.
		Joins("Contact").
		Where(`CAST(product_details AS TEXT) ~ ? OR "Contact".email ~ ? OR "Contact".name ~ ?`, likeText, likeText, likeText)

	messages, err := repository.GetPaginated(
		entity,
		pagination,
		order,
		whereable,
		"",
		db,
	)
	return messages, err
}
