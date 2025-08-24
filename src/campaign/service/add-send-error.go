package campaign_service

import (
	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/google/uuid"
)

func AddSendError(
	campaignMessageID uuid.UUID,
	err error,
) (campaign_entity.CampaignMessageSendError, error) {
	if err == nil {
		return campaign_entity.CampaignMessageSendError{}, nil
	}
	return repository.Create(
		campaign_entity.CampaignMessageSendError{
			CampaignMessageID: campaignMessageID,
			ErrorData:         err.Error(),
		}, database.DB,
	)
}
