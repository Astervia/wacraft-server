package user_service

import (
	"fmt"

	database_model "github.com/Astervia/wacraft-core/src/database/model"
	"github.com/Astervia/wacraft-core/src/repository"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

func ContentKeyLike(
	likeText string,
	key user_model.SearchableUserColumn,
	entity user_entity.User,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]user_entity.User, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	expr := fmt.Sprintf("immutable_unaccent(COALESCE(%s::text, ''))", key)

	// Construct the LIKE query for sender_data, receiver_data, or product_data
	db = db.
		Where(expr+" ILIKE immutable_unaccent(?)", likeText)

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
