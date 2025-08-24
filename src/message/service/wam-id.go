package message_service

import (
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

func GetWamID(
	wamID string,
	entity message_entity.Message,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]message_entity.Message, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	// Construct the specific WHERE clause using JSONB operators
	db = db.
		Joins("From").
		Joins("To").
		Joins("From.Contact").
		Joins("To.Contact").
		Where(
			// Match waID in receiver_data.id
			"receiver_data->>'id' = ? OR "+
				// Match waID in any product_data.messages[].id
				"EXISTS ("+
				"SELECT 1 FROM jsonb_array_elements(product_data->'messages') AS m(message) "+
				"WHERE m.message->>'id' = ?"+
				")",
			wamID,
			wamID,
		)

	// Apply pagination, ordering, and additional where conditions
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
