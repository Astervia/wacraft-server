package user_service

import (
	"fmt"

	database_model "github.com/Astervia/wacraft-core/src/database/model"
	"github.com/Astervia/wacraft-core/src/repository"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

func ContentKeyLike(
	likeText string,
	key string,
	entity user_entity.User,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]user_entity.User, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	db = db.
		Where(
			fmt.Sprintf("CAST(%s AS TEXT) ILIKE ?", key),
			likeText,
		)

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
