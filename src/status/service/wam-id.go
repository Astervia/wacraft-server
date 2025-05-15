package status_service

import (
	"github.com/Astervia/wacraft-server/src/database"
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	"github.com/Astervia/wacraft-core/src/repository"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	"gorm.io/gorm"
)

func GetWamId(
	wamId string,
	entity status_entity.Status,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]status_entity.Status, error) {
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
			// Match waId in receiver_data.id
			"receiver_data->>'id' = ? OR "+
				// Match waId in any product_data.Statuss[].id
				"EXISTS ("+
				"SELECT 1 FROM jsonb_array_elements(product_data->'Statuss') AS m(Status) "+
				"WHERE m.Status->>'id' = ?"+
				")",
			wamId,
			wamId,
		)

	// Apply pagination, ordering, and additional where conditions
	Statuss, err := repository.GetPaginated(
		entity,
		pagination,
		order,
		whereable,
		"",
		db,
	)
	return Statuss, err
}
