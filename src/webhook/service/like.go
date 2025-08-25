package webhook_service

import (
	"fmt"

	database_model "github.com/Astervia/wacraft-core/src/database/model"
	"github.com/Astervia/wacraft-core/src/repository"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_model "github.com/Astervia/wacraft-core/src/webhook/model"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

func ContentKeyLike(
	likeText string,
	key webhook_model.SearchableWebhookColumn,
	entity webhook_entity.Webhook,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]webhook_entity.Webhook, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	// Construct the LIKE query for sender_data, receiver_data, or product_data
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
