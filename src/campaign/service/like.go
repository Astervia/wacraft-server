package campaign_service

import (
	"fmt"

	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	"github.com/Astervia/wacraft-server/src/database"
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"gorm.io/gorm"
)

func ContentKeyLike(
	likeText string,
	key string,
	entity campaign_entity.Campaign,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]campaign_entity.Campaign, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	// Construct the LIKE query for sender_data, receiver_data, or product_data
	db = db.
		Where(
			fmt.Sprintf("CAST(%s AS TEXT) ~ ?", key),
			likeText,
		)

	entities, err := repository.GetPaginated(
		entity,
		pagination,
		order,
		whereable,
		"",
		db,
	)
	return entities, err
}
