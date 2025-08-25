package campaign_service

import (
	"fmt"

	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

func ContentKeyLike(
	likeText string,
	key campaign_model.SearchableCampaignColumn,
	entity campaign_entity.Campaign,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]campaign_entity.Campaign, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	expr := fmt.Sprintf("immutable_unaccent(COALESCE(%s::text, ''))", key)

	// Construct the LIKE query for sender_data, receiver_data, or product_data
	db = db.
		Where(expr+" ILIKE immutable_unaccent(?)", likeText)

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
