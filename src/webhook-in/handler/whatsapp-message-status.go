package webhook_handler

import (
	"sync"

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
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// statusSynchronizer serialises concurrent status updates for the same wamID.
// Uses the DistributedLock from the webhook service package, which defaults
// to in-memory and is replaced with a Redis lock when SYNC_BACKEND=redis.
var statusSynchronizer = whk_service.GetStatusLock()

// Returns status updates from unblocked contacts
func handleStatuses(
	value wh_model.Value, tx *gorm.DB, mpID uuid.UUID,
) ([]status_entity.Status, error) {
	var statuses []status_entity.Status
	var statMu sync.Mutex
	var eg errgroup.Group

	for _, status := range *value.Statuses {
		eg.Go(func() error {
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
				return err
			}
			var msgID uuid.UUID
			if len(msgs) == 0 {
				msgID, err = message_service.StatusSynchronizer.AddStatus(
					wamID,
					status.Status,
					env.MessageStatusSyncTimeout,
				)
				statusSynchronizer.Unlock(wamID)
				if err != nil {
					// Err adding status means that the message will not be added and is irreversible. Must not return error to WhatsApp API
					// This is important to avoid creating unnecessary connections to the database. And for saving resources in general.
					return nil
				}
			} else {
				statusSynchronizer.Unlock(wamID)
				msg := msgs[0]

				blocked := false
				if msg.From.ID != uuid.Nil {
					blocked = msg.From.Blocked
				} else if msg.To.ID != uuid.Nil {
					blocked = msg.To.Blocked
				}
				if blocked {
					return nil
				}
				msgID = msg.ID
			}

			statMu.Lock()
			statuses = append(statuses, status_entity.Status{
				StatusFields: status_model.StatusFields{
					MessageID: msgID,
					ProductData: &status_model.ProductData{
						Status: &status,
					},
				},
			})
			statMu.Unlock()
			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		return nil, err
	}

	// ⚡ BOLT OPTIMIZATION: Batch insert statuses to avoid N+1 queries
	if len(statuses) > 0 {
		if err := tx.Create(&statuses).Error; err != nil {
			return nil, err
		}
	}

	return statuses, nil
}
