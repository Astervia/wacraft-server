package message_service

import (
	"errors"
	"fmt"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	common_service "github.com/Astervia/wacraft-server/src/common/service"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	phone_config_service "github.com/Astervia/wacraft-server/src/phone-config/service"
	bootstrap_module "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap"
	message_service "github.com/Rfluid/whatsapp-cloud-api/src/message"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindMessagingProductByWorkspaceAndSendMessage finds the messaging product for a workspace and sends a message.
// Uses the PhoneConfig associated with the messaging product.
func FindMessagingProductByWorkspaceAndSendMessage(
	body message_model.SendWhatsAppMessage,
	workspaceID uuid.UUID,
	propagateCallback func(message_entity.Message),
) (message_entity.Message, error) {
	mp := messaging_product_entity.MessagingProduct{
		Name:        messaging_product_model.WhatsApp,
		WorkspaceID: &workspaceID,
	}
	err := database.DB.Model(&mp).Where(&mp).First(&mp).Error
	if err != nil {
		return message_entity.Message{}, fmt.Errorf("no messaging product found for workspace: %w", err)
	}

	if mp.PhoneConfigID == nil {
		return message_entity.Message{}, errors.New("messaging product has no phone config configured")
	}

	// Get WhatsApp API from phone config
	wabaApi, err := phone_config_service.GetWhatsAppAPIByPhoneConfigID(*mp.PhoneConfigID)
	if err != nil {
		return message_entity.Message{}, err
	}

	var msg message_entity.Message
	if common_service.IsEnvLocal() {
		msg, err = SendWhatsAppMessageWithAPIWithoutWaitingForStatus(body, mp.ID, wabaApi, nil)
	} else {
		msg, err = SendWhatsAppMessageWithAPI(body, mp.ID, wabaApi, nil)
	}
	if err != nil {
		return msg, err
	}

	go propagateCallback(msg)

	return msg, nil
}

// SendWhatsAppMessageWithAPI sends a message using a specific WhatsApp API instance.
func SendWhatsAppMessageWithAPI(
	body message_model.SendWhatsAppMessage,
	messagingProductID uuid.UUID,
	wabaApi *bootstrap_module.WhatsAppAPI,
	tx *gorm.DB,
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

	// Sending message
	response, err := message_service.Send(*wabaApi, body.SenderData)
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

// SendWhatsAppMessageWithAPIWithoutWaitingForStatus sends a message without waiting for status.
func SendWhatsAppMessageWithAPIWithoutWaitingForStatus(
	body message_model.SendWhatsAppMessage,
	messagingProductID uuid.UUID,
	wabaApi *bootstrap_module.WhatsAppAPI,
	tx *gorm.DB,
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

	// Sending message
	response, err := message_service.Send(*wabaApi, body.SenderData)
	if err != nil {
		return message, err
	}

	message.ProductData = &message_model.ProductData{
		Response: &response,
	}
	if len(message.ProductData.Messages) == 0 {
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
