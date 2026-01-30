package message_service

import (
	"errors"
	"fmt"

	database_model "github.com/Astervia/wacraft-core/src/database/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/integration/whatsapp"
	phone_config_service "github.com/Astervia/wacraft-server/src/phone-config/service"
	bootstrap_module "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap"
	common_model "github.com/Rfluid/whatsapp-cloud-api/src/common"
	message_model "github.com/Rfluid/whatsapp-cloud-api/src/message"
	typing_model "github.com/Rfluid/whatsapp-cloud-api/src/typing"
	typing_service "github.com/Rfluid/whatsapp-cloud-api/src/typing"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SendTypingToUser sends a typing indicator using the legacy global API.
func SendTypingToUser(
	entity message_entity.Message,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	prefix string,
	db *gorm.DB,
) (common_model.SuccessResponse, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	mp := messaging_product_entity.MessagingProduct{Name: messaging_product_model.WhatsApp}

	if err := database.DB.Model(&mp).Where(&mp).First(&mp).Error; err != nil {
		return common_model.SuccessResponse{Success: false}, err
	}
	entity.MessagingProductID = mp.ID

	messages, err := repository.GetPaginated(entity, pagination, order, whereable, prefix, db)
	if err != nil {
		return common_model.SuccessResponse{Success: false}, err
	}

	if len(messages) == 0 {
		return common_model.SuccessResponse{Success: false}, errors.New("message not found")
	}

	msg := messages[0]
	if msg.ReceiverData == nil {
		return common_model.SuccessResponse{Success: false}, errors.New("receiver data not found for latest message")
	}
	msgWamID := msg.ReceiverData.ID
	response, err := typing_service.SendTyping(
		whatsapp.WabaApi,
		typing_model.SendTypingPayload{
			MessageID:        msgWamID,
			Status:           message_model.Read,
			MessagingProduct: common_model.MessagingProduct{MessagingProduct: "whatsapp"},
			TypingIndicator: typing_model.TypingIndicator{
				Type: typing_model.Text,
			},
		},
	)

	return response, err
}

// SendTypingToUserByWorkspace sends a typing indicator using workspace-specific API.
func SendTypingToUserByWorkspace(
	entity message_entity.Message,
	workspaceID uuid.UUID,
	pagination database_model.Paginable,
	order database_model.Orderable,
	whereable database_model.Whereable,
	prefix string,
	db *gorm.DB,
) (common_model.SuccessResponse, error) {
	if db == nil {
		db = database.DB.Model(&entity)
	}

	// Find messaging product for workspace
	mp := messaging_product_entity.MessagingProduct{
		Name:        messaging_product_model.WhatsApp,
		WorkspaceID: &workspaceID,
	}

	if err := database.DB.Model(&mp).Where(&mp).First(&mp).Error; err != nil {
		return common_model.SuccessResponse{Success: false}, fmt.Errorf("no messaging product found for workspace: %w", err)
	}
	entity.MessagingProductID = mp.ID

	// Get appropriate API
	var wabaApi *bootstrap_module.WhatsAppAPI
	if mp.PhoneConfigID != nil {
		var err error
		wabaApi, err = phone_config_service.GetWhatsAppAPIByPhoneConfigID(*mp.PhoneConfigID)
		if err != nil {
			return common_model.SuccessResponse{Success: false}, fmt.Errorf("failed to get WhatsApp API: %w", err)
		}
	} else {
		wabaApi = &whatsapp.WabaApi
	}

	messages, err := repository.GetPaginated(entity, pagination, order, whereable, prefix, db)
	if err != nil {
		return common_model.SuccessResponse{Success: false}, err
	}

	if len(messages) == 0 {
		return common_model.SuccessResponse{Success: false}, errors.New("message not found")
	}

	msg := messages[0]
	if msg.ReceiverData == nil {
		return common_model.SuccessResponse{Success: false}, errors.New("receiver data not found for latest message")
	}
	msgWamID := msg.ReceiverData.ID
	response, err := typing_service.SendTyping(
		*wabaApi,
		typing_model.SendTypingPayload{
			MessageID:        msgWamID,
			Status:           message_model.Read,
			MessagingProduct: common_model.MessagingProduct{MessagingProduct: "whatsapp"},
			TypingIndicator: typing_model.TypingIndicator{
				Type: typing_model.Text,
			},
		},
	)

	return response, err
}
