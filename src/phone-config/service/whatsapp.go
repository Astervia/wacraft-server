package phone_config_service

import (
	"net/http"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	bootstrap_module "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap"
	"github.com/google/uuid"
)

// apiCache caches WhatsApp API instances by phone config ID
// var apiCache = struct {
// 	sync.RWMutex
// 	apis map[uuid.UUID]*bootstrap_module.WhatsAppAPI
// }{apis: make(map[uuid.UUID]*bootstrap_module.WhatsAppAPI)}

var sharedHTTPClient = &http.Client{}

// // GetWhatsAppAPIByPhoneConfigID returns a WhatsApp API instance for the given phone config ID.
// // It caches the API instance for reuse.
// func GetWhatsAppAPIByPhoneConfigID(phoneConfigID uuid.UUID) (*bootstrap_module.WhatsAppAPI, error) {
// 	// Check cache first
// 	apiCache.RLock()
// 	if api, ok := apiCache.apis[phoneConfigID]; ok {
// 		apiCache.RUnlock()
// 		return api, nil
// 	}
// 	apiCache.RUnlock()

// 	// Load phone config from database
// 	var phoneConfig phone_config_entity.PhoneConfig
// 	if err := database.DB.First(&phoneConfig, phoneConfigID).Error; err != nil {
// 		return nil, fmt.Errorf("phone config not found: %w", err)
// 	}

// 	return buildWhatsAppAPI(&phoneConfig)
// }

// GetWhatsAppAPIByWabaID returns a WhatsApp API instance for the given WABA ID (Phone Number ID).
// func GetWhatsAppAPIByWabaID(wabaID string) (*bootstrap_module.WhatsAppAPI, *phone_config_entity.PhoneConfig, error) {
// 	var phoneConfig phone_config_entity.PhoneConfig
// 	if err := database.DB.Where("waba_id = ? AND is_active = true", wabaID).First(&phoneConfig).Error; err != nil {
// 		return nil, nil, fmt.Errorf("phone config not found for waba_id %s: %w", wabaID, err)
// 	}

// 	api, err := buildWhatsAppAPI(&phoneConfig)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	return api, &phoneConfig, nil
// }

// buildWhatsAppAPI creates and caches a WhatsApp API instance from a phone config.
func BuildWhatsAppAPI(phoneConfig *phone_config_entity.PhoneConfig) (*bootstrap_module.WhatsAppAPI, error) {
	version := "v24.0"
	cfg := bootstrap_module.SenderConfig{
		AccessToken:   phoneConfig.AccessToken,
		WABAID:        phoneConfig.WabaID,
		WABAAccountID: phoneConfig.WabaAccountID,
		Version:       &version,
	}
	wabaApi, err := bootstrap_module.FromConfigWithClient(cfg, sharedHTTPClient)

	return wabaApi, err
}

func GetWhatsAppAPIByPhoneConfigID(configID uuid.UUID) (*bootstrap_module.WhatsAppAPI, error) {
	phone_config, err := repository.First(
		phone_config_entity.PhoneConfig{
			Audit: common_model.Audit{ID: configID},
		},
		0, nil, nil, "", database.DB,
	)
	if err != nil {
		return nil, err
	}
	return BuildWhatsAppAPI(&phone_config)
}

func GetFirstConfigAPIByWorkspace(workspaceID uuid.UUID) (*bootstrap_module.WhatsAppAPI, error) {
	phone_config, err := repository.First(
		phone_config_entity.PhoneConfig{
			WorkspaceID: &workspaceID,
		},
		0, nil, nil, "", database.DB,
	)
	if err != nil {
		return nil, err
	}
	return BuildWhatsAppAPI(&phone_config)
}

// InvalidateCache removes a phone config from the cache.
// Call this when a phone config is updated or deleted.
// func InvalidateCache(phoneConfigID uuid.UUID) {
// 	apiCache.Lock()
// 	delete(apiCache.apis, phoneConfigID)
// 	apiCache.Unlock()
// }

// ClearCache removes all cached API instances.
// func ClearCache() {
// 	apiCache.Lock()
// 	apiCache.apis = make(map[uuid.UUID]*bootstrap_module.WhatsAppAPI)
// 	apiCache.Unlock()
// }

// GetMetaAppSecret returns the Meta app secret for signature verification.
func GetMetaAppSecret(phoneConfig *phone_config_entity.PhoneConfig) string {
	return phoneConfig.MetaAppSecret
}

// GetVerifyToken returns the webhook verify token.
func GetVerifyToken(phoneConfig *phone_config_entity.PhoneConfig) string {
	return phoneConfig.WebhookVerifyToken
}
