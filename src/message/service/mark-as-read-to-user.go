package message_service

import (
	"errors"

	database_model "github.com/Astervia/wacraft-core/src/database/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/integration/whatsapp"
	common_model "github.com/Rfluid/whatsapp-cloud-api/src/common/model"
	message_model "github.com/Rfluid/whatsapp-cloud-api/src/message/model"
	message_service "github.com/Rfluid/whatsapp-cloud-api/src/message/service"
	"gorm.io/gorm"
)

func MarkWhatsAppMessageAsReadToUser(
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
	response, err := message_service.MarkAsRead(
		whatsapp.WabaApi,
		message_model.MarkAsRead{
			MessageID:        msgWamID,
			Status:           message_model.Read,
			MessagingProduct: common_model.MessagingProduct{MessagingProduct: "whatsapp"},
		},
	)

	return response, err
}
