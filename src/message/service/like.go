package message_service

import (
	"fmt"

	database_model "github.com/Astervia/wacraft-core/src/database/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

// Query for messages with a specific content checking if sender_data, receiver_data, or product_data contains the likeText.
func ContentLike(
	likeText string,
	entity message_entity.Message,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]message_entity.Message, error) {
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

func ContentKeyLike(
	likeText string,
	key string,
	entity message_entity.Message,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]message_entity.Message, error) {
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
			fmt.Sprintf("CAST(%s AS TEXT) ~ ?", string(key)),
			likeText,
		)

	messages, err := repository.GetPaginated(
		entity,
		pagination,
		order,
		whereable,
		"messages",
		db,
	)
	return messages, err
}

// Query for messages with a specific content checking if sender_data, receiver_data, or product_data contains the likeText.
func CountContentLike(
	likeText string,
	entity message_entity.Message,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) (int64, error) {
	if db == nil {
		db = database.DB
	}

	// Construct the LIKE query for sender_data, receiver_data, or product_data
	db = db.
		Joins("From").
		Joins("To").
		Joins("From.Contact").
		Joins("To.Contact").
		Where(`CAST(sender_data AS TEXT) ~ ? OR CAST(receiver_data AS TEXT) ~ ? OR CAST(product_data AS TEXT) ~ ?`, likeText, likeText, likeText)

	messages, err := repository.Count(
		entity,
		order,
		whereable,
		"",
		db,
	)
	return messages, err
}
