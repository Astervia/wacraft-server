package messaging_product_service

import (
	"fmt"

	database_model "github.com/Astervia/wacraft-core/src/database/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

// Query product_details (jsonb) + related Contact (email/name) using trigram + unaccent.
// IMPORTANT: This matches the indexes created (immutable_unaccent + COALESCE).
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

	// Expressions that mirror index definitions
	const prodExpr = "immutable_unaccent(COALESCE(product_details::text, ''))"
	const emailExpr = `immutable_unaccent(COALESCE("Contact".email, ''))`
	const nameExpr = `immutable_unaccent(COALESCE("Contact".name, ''))`

	q := db.
		Joins("Contact").
		Where(
			fmt.Sprintf("%s ILIKE immutable_unaccent(?) OR %s ILIKE immutable_unaccent(?) OR %s ILIKE immutable_unaccent(?)",
				prodExpr, emailExpr, nameExpr),
			likeText, likeText, likeText,
		)

	return repository.GetPaginated(
		entity,
		pagination,
		order,
		whereable,
		"",
		q,
	)
}
