package webhook_handler

import (
	"fmt"
	"sync"

	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_out_model "github.com/Astervia/wacraft-core/src/webhook/model"
	"github.com/Astervia/wacraft-server/src/database"
	message_handler "github.com/Astervia/wacraft-server/src/message/handler"
	messaging_product_service "github.com/Astervia/wacraft-server/src/messaging-product/service"
	status_handler "github.com/Astervia/wacraft-server/src/status/handler"
	webhook_service "github.com/Astervia/wacraft-server/src/webhook/service"
	wh_model "github.com/Rfluid/whatsapp-cloud-api/src/webhook/model"
	webhook_model "github.com/Rfluid/whatsapp-webhook-server/src/webhook/model"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

var messageExecutionContexts = []wh_model.Field{wh_model.Messages}

var MessageHandler = webhook_model.ChangeHandler{
	Callback:          messageCallback,
	ExecutionContexts: &messageExecutionContexts,
}

func messageCallback(ctx *fiber.Ctx, body *wh_model.WebhookBody, change *wh_model.Change) error {
	if change.Value.Messages == nil && change.Value.Statuses == nil {
		return nil
	}
	var eg errgroup.Group
	var msgs []message_entity.Message
	var statuses []status_entity.Status

	mp := messaging_product_entity.MessagingProduct{Name: messaging_product_model.WhatsApp}

	if err := database.DB.Model(&mp).Where(&mp).First(&mp).Error; err != nil {
		return err
	}

	// Begin transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	value := change.Value

	eg.Go(func() error {
		if value.Messages != nil {
			var err error
			msgs, err = handleMessages(value, tx, mp.ID)
			return err
		}
		return nil
	})

	eg.Go(func() error {
		if value.Statuses != nil {
			var err error
			statuses, err = handleStatuses(value, tx, mp.ID)
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		pterm.DefaultLogger.Error(
			fmt.Sprintf("Error while handling message: %s", err.Error()),
		)
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	go func() {
		for _, msg := range msgs {
			go message_handler.NewMessageChannel.BroadcastJsonMultithread(msg)
			go webhook_service.SendAllByQuery(
				webhook_entity.Webhook{
					Event: webhook_out_model.ReceiveWhatsAppMessage,
				},
				msg,
			)
		}
	}()

	go func() {
		for _, status := range statuses {
			go status_handler.NewStatusChannel.BroadcastJsonMultithread(status)
		}
	}()

	return nil
}

// Returns messages from unblocked contacts
func handleMessages(value wh_model.Value, tx *gorm.DB, mpID uuid.UUID) ([]message_entity.Message, error) {
	var eg errgroup.Group

	msgs := []message_entity.Message{}
	var msgsMu sync.Mutex

	// Handling each message
	for index, message := range *value.Messages {
		eg.Go(func() error {
			// Interpolating message properties
			var name string
			if value.Contacts != nil && len(*value.Contacts) >= index {
				name = (*value.Contacts)[index].Profile.Name
			}

			mpContact, err := messaging_product_service.GetContactOrSave(
				messaging_product_entity.MessagingProductContact{
					MessagingProductID: mpID,
					ProductDetails: &messaging_product_model.ProductDetails{
						WhatsAppProductDetails: &messaging_product_model.WhatsAppProductDetails{
							WaID:        message.From,
							PhoneNumber: message.From,
						},
					},
				},
				contact_entity.Contact{
					Name:  &name,
					Email: nil,
				},
				tx,
			)
			if err != nil {
				return err
			}

			// Building the message entity and creating with the mp contact found
			msg := message_entity.Message{
				MessageFields: message_model.MessageFields{
					ReceiverData:       &message_model.ReceiverData{MessageReceived: &message},
					FromID:             &mpContact.ID,
					MessagingProductID: mpID,
				},
				From: &mpContact,
			}
			if msg.From.Blocked {
				return nil
			}
			err = tx.Model(&msg).Create(&msg).Error
			if err != nil {
				return err
			}
			msgsMu.Lock()
			msgs = append(msgs, msg)
			msgsMu.Unlock()
			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		pterm.DefaultLogger.Error(
			fmt.Sprintf("Error while handling message: %s", err.Error()),
		)
	}

	return msgs, err
}
