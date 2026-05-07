package webhook_handler

import (
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	status_model "github.com/Astervia/wacraft-core/src/status/model"
	"github.com/Astervia/wacraft-server/src/config/env"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	whk_service "github.com/Astervia/wacraft-server/src/webhook-in/service"
	wh_model "github.com/Rfluid/whatsapp-cloud-api/src/webhook"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// statusSynchronizer serialises concurrent status updates for the same wamID.
// Uses the DistributedLock from the webhook service package, which defaults
// to in-memory and is replaced with a Redis lock when SYNC_BACKEND=redis.
var statusSynchronizer = whk_service.GetStatusLock()

// Returns status updates from unblocked contacts.
//
// Processed sequentially so the shared *gorm.DB transaction is used by a
// single goroutine. Status rows are accumulated and inserted in a single
// batch via tx.Create(&statuses) to avoid the previous N+1 insert pattern.
func handleStatuses(
	value wh_model.Value, tx *gorm.DB, mpID uuid.UUID,
) ([]status_entity.Status, error) {
	statuses := make([]status_entity.Status, 0, len(*value.Statuses))

	for _, status := range *value.Statuses {
		ascending := database_model.Asc
		wamID := status.ID

		statusSynchronizer.Lock(wamID)

		msgs, err := message_service.GetWamID(
			wamID,
			message_entity.Message{
				MessageFields: message_model.MessageFields{
					MessagingProductID: mpID,
				},
			},
			&database_model.Paginate{
				Offset: 0,
				Limit:  1,
			},
			&database_model.DateOrder{
				CreatedAt: &ascending,
			},
			nil,
			tx,
		)
		if err != nil {
			statusSynchronizer.Unlock(wamID)
			return statuses, err
		}

		var msgID uuid.UUID
		if len(msgs) == 0 {
			_, addErr := message_service.StatusSynchronizer.AddStatus(
				wamID,
				status.Status,
				env.MessageStatusSyncTimeout,
			)
			statusSynchronizer.Unlock(wamID)
			if addErr != nil {
				// Failure to register a deferred status is irreversible: the
				// message row does not yet exist. Skip silently to avoid
				// returning an error to the WhatsApp API and to limit DB
				// connection churn under load.
				continue
			}
			// No row to insert — the status will be applied later when the
			// matching message is saved.
			continue
		}

		statusSynchronizer.Unlock(wamID)
		msg := msgs[0]

		blocked := false
		if msg.From.ID != uuid.Nil {
			blocked = msg.From.Blocked
		} else if msg.To.ID != uuid.Nil {
			blocked = msg.To.Blocked
		}
		if blocked {
			continue
		}
		msgID = msg.ID

		statuses = append(statuses, status_entity.Status{
			StatusFields: status_model.StatusFields{
				MessageID: msgID,
				ProductData: &status_model.ProductData{
					Status: &status,
				},
			},
		})
	}

	if len(statuses) > 0 {
		if err := tx.Create(&statuses).Error; err != nil {
			return statuses, err
		}
	}

	return statuses, nil
}
