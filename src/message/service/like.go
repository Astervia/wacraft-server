package message_service

import (
	"fmt"

	database_model "github.com/Astervia/wacraft-core/src/database/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"gorm.io/gorm"
)

// Expression MUST match the functional GIN index:
// idx_messages_search_trgm_unaccent ON immutable_unaccent(
//
//	COALESCE(sender_data::text, '') || ' ' || COALESCE(receiver_data::text, '') || ' ' || COALESCE(product_data::text, '')
//
// ) gin_trgm_ops
const searchableExpr = "immutable_unaccent((" +
	"COALESCE(sender_data::text, '') || ' ' || " +
	"COALESCE(receiver_data::text, '') || ' ' || " +
	"COALESCE(product_data::text, '')" +
	"))"

// Query for messages with content across sender/receiver/product using trigram GIN.
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

	db = db.
		Joins("From").
		Joins("To").
		Joins("From.Contact").
		Joins("To.Contact").
		// IMPORTANT: apply immutable_unaccent to BOTH sides so the index can be used
		Where(searchableExpr+" ILIKE immutable_unaccent(?)", likeText)

	return repository.GetPaginated(entity, pagination, order, whereable, "", db)
}

// Query for messages with content on a specific column (sender_data / receiver_data / product_data).
func ContentKeyLike(
	pattern string, // caller may pass "%term%" or we can build it externally
	key message_model.JsonMessageKey,
	entity message_entity.Message,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	db *gorm.DB,
) ([]message_entity.Message, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	// Build expression: immutable_unaccent(COALESCE(<key>::text,''))
	expr := fmt.Sprintf("immutable_unaccent(COALESCE(%s::text, ''))", key)

	db = db.
		Joins("From").
		Joins("To").
		Joins("From.Contact").
		Joins("To.Contact").
		Where(expr+" ILIKE immutable_unaccent(?)", pattern)

	return repository.GetPaginated(entity, pagination, order, whereable, "messages", db)
}

// Count version of ContentLike (uses the same indexed expression)
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

	db = db.
		Joins("From").
		Joins("To").
		Joins("From.Contact").
		Joins("To.Contact").
		Where(searchableExpr+" ILIKE immutable_unaccent(?)", likeText)

	return repository.Count(entity, order, whereable, "", db)
}
