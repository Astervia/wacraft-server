package status_service

import (
	"errors"
	"fmt"
	"strings"

	database_model "github.com/Astervia/wacraft-core/src/database/model"
	"github.com/Astervia/wacraft-core/src/repository"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	status_model "github.com/Astervia/wacraft-core/src/status/model"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

// Query for statuses with a specific content checking if sender_data, receiver_data, or product_data contains the likeText.
func ContentLike(
	likeText string,
	entity status_entity.Status,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]status_entity.Status, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	// Construct the LIKE query for sender_data, receiver_data, or product_data
	db = db.
		Joins("From").
		Joins("To").
		Joins("From.Contact").
		Joins("To.Contact").
		Where(`CAST(sender_data AS TEXT) ~ ? OR CAST(receiver_data AS TEXT) ~ ? OR CAST(product_data AS TEXT) ~ ?`, likeText, likeText, likeText)

	statuses, err := repository.GetPaginated(
		entity,
		pagination,
		order,
		whereable,
		"",
		db,
	)
	return statuses, err
}

func ContentKeyLike(
	likeText string,
	key status_model.SearchableStatusColumn,
	entity status_entity.Status,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]status_entity.Status, error) {
	normalizedKey := status_model.SearchableStatusColumn(strings.ToLower(string(key)))
	if !normalizedKey.IsValid() {
		return nil, errors.New("invalid status search key")
	}

	if db == nil {
		db = database.DB.Model(&entity)
	}

	// Construct the LIKE query for sender_data, receiver_data, or product_data
	db = db.
		Joins("From").
		Joins("To").
		Joins("From.Contact").
		Joins("To.Contact").
		Where(
			fmt.Sprintf("CAST(%s AS TEXT) ~ ?", normalizedKey),
			likeText,
		)

	statuses, err := repository.GetPaginated(
		entity,
		pagination,
		order,
		whereable,
		"statuses",
		db,
	)
	return statuses, err
}
