package messaging_product_service

import (
	"fmt"

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
		db = database.DB.Model(&entity)
	}

	// Expressions that mirror index definitions
	const prodExpr = "immutable_unaccent(COALESCE(product_details::text, ''))"
	const emailExpr = `immutable_unaccent(COALESCE("Contact".email, ''))`
	const nameExpr = `immutable_unaccent(COALESCE("Contact".name, ''))`

	db = db.
		Joins("Contact").
		Where(
			fmt.Sprintf("%s ILIKE immutable_unaccent(?) OR %s ILIKE immutable_unaccent(?) OR %s ILIKE immutable_unaccent(?)",
				prodExpr, emailExpr, nameExpr),
			likeText, likeText, likeText,
		)

	c, err := repository.Count(
		entity,
		order,
		whereable,
		"",
		db,
	)
	return c, err
}
