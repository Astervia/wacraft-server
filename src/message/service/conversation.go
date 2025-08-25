package message_service

import (
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetConversation(
	mpcID uuid.UUID, // ID of messaging product contact
	entity message_entity.Message,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	prefix string,
	db *gorm.DB,
) ([]message_entity.Message, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}
	// Add the condition to filter messages sent or received by the messaging product contact
	db = db.
		Where("from_id = ? OR to_id = ?", mpcID, mpcID).
		Preload("Statuses", func(db *gorm.DB) *gorm.DB {
			return db.
				Select("DISTINCT ON (message_id) *").
				Order("message_id").
				Order(`
                    CASE statuses.product_data->>'status'
                        WHEN 'read' THEN 1
                        WHEN 'delivered' THEN 2
                        WHEN 'sent' THEN 3
                        ELSE 0
                    END ASC
                `)
		})

	return repository.GetPaginated(
		entity, pagination, order, whereable, prefix, db,
	)
}

func GetLatestMessagesForEachUser(
	message message_entity.Message,
	paginable database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]message_entity.Message, error) {
	var messages []message_entity.Message
	if db == nil {
		db = database.DB
	}

	subquery := db.
		Model(&message_entity.Message{}).
		Select("DISTINCT ON (COALESCE(from_id, to_id)) COALESCE(from_id, to_id) AS contact_id, *").
		// Joins("From").
		// Joins("To").
		// Joins("From.Contact").
		// Joins("To.Contact").
		Where(&message).
		Order("COALESCE(from_id, to_id)").
		Order(`"messages".created_at DESC`)

	db = db.
		Table("(?) AS sq", subquery).
		Joins("From").
		Joins("To").
		Joins("From.Contact").
		Joins("To.Contact").
		Preload("Statuses", func(db *gorm.DB) *gorm.DB {
			return db.
				Select("DISTINCT ON (message_id) *").
				Order("message_id").
				Order(`
                    CASE statuses.product_data->>'status'
                        WHEN 'read' THEN 1
                        WHEN 'delivered' THEN 2
                        WHEN 'sent' THEN 3
                        ELSE 0
                    END ASC
                `)
		}).
		Select("sq.*")

	if paginable != nil {
		paginable.PaginateQuery(&db)
	}

	if order != nil {
		order.OrderQuery(&db, `"sq"`)
	}
	if whereable != nil {
		whereable.Where(&db, `"sq"`)
	}

	err := db.Find(&messages).Error

	return messages, err
}

func CountConversations(
	mpcID uuid.UUID, // ID of messaging product contact
	entity message_entity.Message,
	order database_model.Orderable,
	whereable database_model.Whereable,
	prefix string,
	db *gorm.DB,
) (int64, error) {
	if db == nil {
		db = database.DB
	}
	// Add the condition to filter messages sent or received by the messaging product contact
	db = db.Where("from_id = ? OR to_id = ?", mpcID, mpcID)

	return repository.Count(entity, order, whereable, prefix, db)
}

func CountDistinctConversations(
	entity message_entity.Message,
	order database_model.Orderable,
	whereable database_model.Whereable,
	prefix string,
	db *gorm.DB,
) (int64, error) {
	if db == nil {
		db = database.DB
	}
	// Add the condition to filter messages sent or received by the messaging product contact
	db = db.
		Group("COALESCE(from_id, to_id)")

	return repository.Count(entity, order, whereable, prefix, db)
}

// Query for messages with a specific content checking if sender_data, receiver_data, or product_data contains the likeText.
func ConversationContentLike(
	mpcID uuid.UUID, // ID of messaging product contact
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
		Preload("Statuses", func(db *gorm.DB) *gorm.DB {
			return db.
				Select("DISTINCT ON (message_id) *").
				Order("message_id").
				Order(`
                    CASE statuses.product_data->>'status'
                        WHEN 'read' THEN 1
                        WHEN 'delivered' THEN 2
                        WHEN 'sent' THEN 3
                        ELSE 0
                    END ASC
                `)
		}).
		Where(searchableExpr+" ILIKE immutable_unaccent(?)", likeText).
		Where("from_id = ? OR to_id = ?", mpcID, mpcID)

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

func CountConversationContentLike(
	mpcID uuid.UUID, // ID of messaging product contact
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
		Where(searchableExpr+" ILIKE immutable_unaccent(?)", likeText).
		Where("from_id = ? OR to_id = ?", mpcID, mpcID)

	messages, err := repository.Count(
		entity,
		order,
		whereable,
		"",
		db,
	)
	return messages, err
}
