package campaign_service

import (
	"context"
	"errors"
	"sync"

	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	"github.com/Astervia/wacraft-server/src/database"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	message_handler "github.com/Astervia/wacraft-server/src/message/handler"
	message_model "github.com/Astervia/wacraft-core/src/message/model"
	message_service "github.com/Astervia/wacraft-server/src/message/service"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	messaging_product_service "github.com/Astervia/wacraft-server/src/messaging-product/service"
	"github.com/Astervia/wacraft-core/src/repository"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_model "github.com/Astervia/wacraft-core/src/webhook/model"
	webhook_service "github.com/Astervia/wacraft-server/src/webhook/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SendWhatsAppCampaign(
	campaignId uuid.UUID,
	campaignChannel campaign_model.CampaignChannel,
	callback func(*campaign_model.CampaignResults),
) (campaign_model.CampaignResults, error) {
	campaignChannel.SendingMu.Lock()
	if campaignChannel.Sending {
		return campaign_model.CampaignResults{}, errors.New("campaign is already sending")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is called when the function exits
	campaignChannel.Sending = true
	campaignChannel.AddCancel(&cancel)
	campaignChannel.SendingMu.Unlock()

	defer func() {
		campaignChannel.SendingMu.Lock()
		campaignChannel.Sending = false
		campaignChannel.AddCancel(nil)
		campaignChannel.SendingMu.Unlock()
	}()

	campaign := campaign_entity.Campaign{
		Audit:            common_model.Audit{Id: campaignId},
		MessagingProduct: &messaging_product_entity.MessagingProduct{Name: messaging_product_model.WhatsApp},
	}
	if err := database.DB.Model(&campaign).Where(&campaign).First(&campaign).Error; err != nil {
		return campaign_model.CampaignResults{}, err
	}

	campaignMessage := campaign_entity.CampaignMessage{
		CampaignId: campaignId,
	}

	var messagesCount int64
	// Find campaign
	if err := database.DB.Model(&campaignMessage).Where(&campaignMessage).Where("message_id IS NULL").Count(&messagesCount).Error; err != nil {
		return campaign_model.CampaignResults{}, err
	}
	if messagesCount == 0 {
		return campaign_model.CampaignResults{}, errors.New("no messages to send")
	}

	result := campaign_model.CreateCampaignResults(messagesCount)

	var wg sync.WaitGroup
	errCh := make(chan error, messagesCount)

	offset := 0
	var offsetMu sync.Mutex

	for i := 0; i < int(messagesCount); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				err := SendWhatsAppCampaignMessage(
					campaignId,
					*campaign.MessagingProductId,
					&offset,
					&offsetMu,
				)
				errCh <- err
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		result.HandleError(err, callback)
		// Checks if error is no message to send
		if err != nil && err.Error() == "no messages to send" {
			cancel()
			return *result, err
		}
	}

	return *result, nil
}

var contactSynchronizer *synch_service.MutexSwapper[string] = CreateContactSynchronizer()

// Gets first message not sent at campaign, gets related WhatsApp contact or save and sends message.
func SendWhatsAppCampaignMessage(
	campaignId uuid.UUID,
	messagingProductId uuid.UUID,
	offset *int,
	offsetMu *sync.Mutex,
) error {
	var err error

	tx := database.DB

	campaignMessage := campaign_entity.CampaignMessage{
		CampaignId: campaignId,
	}

	// Getting campaign and incrementing offset
	offsetMu.Lock()
	// Query campaign messages that satisfy the entity and where messageId is null
	err = tx.Where(&campaignMessage).Where("message_id IS NULL").Offset(*offset).First(&campaignMessage).Error
	if err != nil {
		(*offset) = (*offset) + 1
		offsetMu.Unlock()

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("no messages to send")
		}
		return err
	}
	defer func() {
		AddSendError(campaignMessage.Id, err)
	}()

	(*offset) = (*offset) + 1
	offsetMu.Unlock()

	if campaignMessage.SenderData == nil {
		return errors.New("sender data is required")
	}

	senderData := *campaignMessage.SenderData

	// Get or create messaging product contact
	var mpc messaging_product_entity.MessagingProductContact
	contactSynchronizer.Lock(senderData.To)
	mpc, err = messaging_product_service.GetContactOrSave(
		messaging_product_entity.MessagingProductContact{
			MessagingProductId: messagingProductId,
			ProductDetails: &messaging_product_model.ProductDetails{
				WhatsAppProductDetails: &messaging_product_model.WhatsAppProductDetails{
					WaId:        senderData.To,
					PhoneNumber: senderData.To,
				},
			},
		},
		contact_entity.Contact{},
		tx,
	)
	contactSynchronizer.Unlock(senderData.To)
	if err != nil {
		return err
	}

	var msg message_entity.Message
	msg, err = message_service.SendWhatsAppMessageAtTransactionWithoutWaitingForStatus(
		message_model.SendWhatsAppMessage{
			ToId:       mpc.Id,
			SenderData: *senderData.Message,
		},
		messagingProductId,
		tx,
	)
	if err != nil {
		return err
	}
	// Propagating results
	func(data message_entity.Message) {
		go message_handler.NewMessageChannel.BroadcastJsonMultithread(data)
		go webhook_service.SendAllByQuery(
			webhook_entity.Webhook{
				Event: webhook_model.SendWhatsAppMessage,
			},
			data,
		)
	}(msg)

	campaignMessageUpdateData := campaign_entity.CampaignMessage{
		MessageId: msg.Id,
	}

	offsetMu.Lock()
	defer offsetMu.Unlock()

	_, err = repository.Updates(campaignMessageUpdateData, &campaign_entity.CampaignMessage{Audit: common_model.Audit{Id: campaignMessage.Id}},
		tx,
	)
	if err != nil {
		return err
	}

	defer func() {
		(*offset) = (*offset) - 1
	}()

	return nil
}
