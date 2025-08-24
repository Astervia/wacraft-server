package status_service

import (
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	"github.com/Astervia/wacraft-core/src/repository"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

func GetWamID(
	wamID string,
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
			// Match waID in receiver_data.id
			"receiver_data->>'id' = ? OR "+
				// Match waID in any product_data.Statuss[].id
				"EXISTS ("+
				"SELECT 1 FROM jsonb_array_elements(product_data->'Statuss') AS m(Status) "+
				"WHERE m.Status->>'id' = ?"+
				")",
			wamID,
			wamID,
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
