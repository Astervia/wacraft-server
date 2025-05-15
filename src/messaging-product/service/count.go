package messaging_product_service

import (
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

func ContactContentLikeCount(
	likeText string,
	entity messaging_product_entity.MessagingProductContact,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) (int64, error) {
	if db == nil {
		db = database.DB
	}

	db = db.
		Joins("Contact").
		Where(`CAST(product_details AS TEXT) ~ ? OR "Contact".email ~ ? OR "Contact".name ~ ?`, likeText, likeText, likeText)

	c, err := repository.Count(
		entity,
		order,
		whereable,
		"",
		db,
	)
	return c, err
}
