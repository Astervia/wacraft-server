package phone_config_service

import (
	"errors"

	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	bootstrap_module "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap"
)

func GetWorkspaceWhatsAppAPI(workspace *workspace_entity.Workspace) (*bootstrap_module.WhatsAppAPI, error) {
	// Find messaging product for workspace
	mp := messaging_product_entity.MessagingProduct{
		Name:        messaging_product_model.WhatsApp,
		WorkspaceID: &workspace.ID,
	}

	if mp.PhoneConfigID == nil {
		return nil, errors.New("empty phone config ID in messaging product")
	}

	wabaApi, err := GetWhatsAppAPIByPhoneConfigID(*mp.PhoneConfigID)

	// Fallback to global API
	return wabaApi, err
}
