package message_service

import (
	"errors"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	common_service "github.com/Astervia/wacraft-server/src/common/service"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/integration/whatsapp"
	message_service "github.com/Rfluid/whatsapp-cloud-api/src/message/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func FindMessagingProductAndSendMessage(
	body message_model.SendWhatsAppMessage,
	propagateCallback func(message_entity.Message),
) (message_entity.Message, error) {
	mp := messaging_product_entity.MessagingProduct{Name: messaging_product_model.WhatsApp}
	err := database.DB.Model(&mp).Where(&mp).First(&mp).Error
	if err != nil {
		return message_entity.Message{}, err
	}

	var msg message_entity.Message
	if common_service.IsEnvLocal() {
		msg, err = SendWhatsAppMessageAtTransactionWithoutWaitingForStatus(body, mp.ID, nil)
	} else {
		msg, err = SendWhatsAppMessageAtTransaction(body, mp.ID, nil)
	}
	if err != nil {
		return msg, err
	}

	go propagateCallback(msg)

	return msg, nil
}

// Sends whatsapp messages at transaction and adds message to status synchronizer.
// Handles messages saved and rollback.
func SendWhatsAppMessageAtTransaction(
	body message_model.SendWhatsAppMessage,
	messagingProductID uuid.UUID,
	tx *gorm.DB, // Transaction. If nil will be executed without transaction.
) (message_entity.Message, error) {
	var message message_entity.Message
	body.SenderData.SetDefault()
	message.ToID = &body.ToID
	message.MessagingProductID = messagingProductID

	// Begin transaction
	transactionProvided := tx != nil
	if tx == nil {
		tx = database.DB
	}

	// Adding contact to message
	contact := messaging_product_entity.MessagingProductContact{
		Audit:              common_model.Audit{ID: body.ToID},
		MessagingProductID: messagingProductID,
	}
	if err := tx.Model(&contact).Where(&contact).Joins("Contact").First(&contact).Error; err != nil {
		return message, err
	}
	message.To = &contact

	// Building message content
	body.SenderData.To = contact.ProductDetails.PhoneNumber
	message.SenderData = &message_model.SenderData{
		Message: &body.SenderData,
	}

	// Sending messsage
	response, err := message_service.Send(whatsapp.WabaApi, body.SenderData)
	if err != nil {
		return message, err
	}

	message.ProductData = &message_model.ProductData{
		Response: &response,
	}
	if len(message.ProductData.Messages) == 0 {
		return message, errors.New("no message id returned by Meta")
	}

	err = StatusSynchronizer.AddMessage(
		message.ProductData.Messages[0].ID.ID,
		env.MessageStatusSyncTimeout,
	)
	if err != nil {
		return message, err
	}

	// Creating message at database
	err = tx.Create(&message).Error
	if err != nil {
		StatusSynchronizer.RollbackMessage(
			message.ProductData.Messages[0].ID.ID,
			env.MessageStatusSyncTimeout,
		)
		return message, err
	}

	go func() {
		if !transactionProvided {
			StatusSynchronizer.MessageSaved(
				message.ProductData.Messages[0].ID.ID,
				message.ID,
				env.MessageStatusSyncTimeout,
			)
		}
	}()

	return message, nil
}

// Same as above but does not lock at waiting for status
func SendWhatsAppMessageAtTransactionWithoutWaitingForStatus(
	body message_model.SendWhatsAppMessage,
	messagingProductID uuid.UUID,
	tx *gorm.DB, // Transaction. If nil will be executed without transaction.
) (message_entity.Message, error) {
	var message message_entity.Message
	body.SenderData.SetDefault()
	message.ToID = &body.ToID
	message.MessagingProductID = messagingProductID

	// Begin transaction
	transactionProvided := tx != nil
	if tx == nil {
		tx = database.DB
	}

	// Adding contact to message
	contact := messaging_product_entity.MessagingProductContact{
		Audit:              common_model.Audit{ID: body.ToID},
		MessagingProductID: messagingProductID,
	}
	if err := tx.Model(&contact).Where(&contact).Joins("Contact").First(&contact).Error; err != nil {
		return message, err
	}
	message.To = &contact

	// Building message content
	body.SenderData.To = contact.ProductDetails.PhoneNumber
	message.SenderData = &message_model.SenderData{
		Message: &body.SenderData,
	}

	// Sending messsage
	response, err := message_service.Send(whatsapp.WabaApi, body.SenderData)
	if err != nil {
		return message, err
	}

	message.ProductData = &message_model.ProductData{
		Response: &response,
	}
	if message.ProductData.Messages == nil || len(message.ProductData.Messages) == 0 {
		return message, errors.New("no message id returned by Meta")
	}

	addMessageCh := make(chan error)
	go func() {
		addMessageCh <- StatusSynchronizer.AddMessage(
			message.ProductData.Messages[0].ID.ID,
			env.MessageStatusSyncTimeout,
		)
	}()

	// Creating message at database
	err = tx.Create(&message).Error
	if err != nil {
		go func() {
			if <-addMessageCh != nil {
				return
			}
			StatusSynchronizer.RollbackMessage(
				message.ProductData.Messages[0].ID.ID,
				env.MessageStatusSyncTimeout,
			)
		}()
		return message, err
	}

	go func() {
		if !transactionProvided {
			if <-addMessageCh != nil {
				return
			}
			StatusSynchronizer.MessageSaved(
				message.ProductData.Messages[0].ID.ID,
				message.ID,
				env.MessageStatusSyncTimeout,
			)
		} else {
			<-addMessageCh
		}
	}()

	return message, nil
}
