package webhook_handler

import (
	"sync"

	wh_model "github.com/Rfluid/whatsapp-cloud-api/src/webhook/model"
	"github.com/Astervia/wacraft-server/src/config/env"
	database_model "github.com/Astervia/wacraft-core/src/database/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	"github.com/Astervia/wacraft-core/src/repository"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	status_model "github.com/Astervia/wacraft-core/src/status/model"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	whk_service "github.com/Astervia/wacraft-server/src/webhook-in/service"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// Synchronize when two status for the same message come together
var statusSynchronizer *synch_service.MutexSwapper[string] = whk_service.CreateStatusSynchronizer()

// Returns status updates from unblocked contacts
func handleStatuses(
	value wh_model.Value, tx *gorm.DB, mpId uuid.UUID,
) ([]status_entity.Status, error) {
	var statuses []status_entity.Status
	var statMu sync.Mutex
	var eg errgroup.Group

	for _, status := range *value.Statuses {
		eg.Go(func() error {
			ascending := database_model.Asc
			wamId := status.Id

			statusSynchronizer.Lock(wamId)

			msgs, err := message_service.GetWamId(
				wamId,
				message_entity.Message{
					MessageFields: message_model.MessageFields{
						MessagingProductId: mpId,
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
				statusSynchronizer.Unlock(wamId)
				return err
			}
			var msgId uuid.UUID
			if len(msgs) == 0 {
				msgId, err = message_service.StatusSynchronizer.AddStatus(
					wamId,
					status.Status,
					env.MessageStatusSyncTimeout,
				)
				statusSynchronizer.Unlock(wamId)
				if err != nil {
					// Err adding status means that the message will not be added and is irreversible. Must not return error to WhatsApp API
					// This is important to avoid creating unnecessary connections to the database. And for saving resources in general.
					return nil
				}
			} else {
				statusSynchronizer.Unlock(wamId)
				msg := msgs[0]

				blocked := false
				if msg.From.Id != uuid.Nil {
					blocked = msg.From.Blocked
				} else if msg.To.Id != uuid.Nil {
					blocked = msg.To.Blocked
				}
				if blocked {
					return nil
				}
				msgId = msg.Id
			}

			s, err := repository.Create(
				status_entity.Status{
					StatusFields: status_model.StatusFields{
						MessageId: msgId,
						ProductData: &status_model.ProductData{
							Status: &status,
						},
					},
				},
				tx,
			)
			if err != nil {
				return err
			}
			statMu.Lock()
			statuses = append(statuses, s)
			statMu.Unlock()
			return nil
		})
	}

	err := eg.Wait()

	return statuses, err
}
