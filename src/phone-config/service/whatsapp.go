package phone_config_service

import (
	"fmt"
	"sync"

	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	"github.com/Astervia/wacraft-server/src/database"
	bootstrap_module "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap"
	bootstrap_service "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap"
	"github.com/google/uuid"
)

// apiCache caches WhatsApp API instances by phone config ID
var apiCache = struct {
	sync.RWMutex
	apis map[uuid.UUID]*bootstrap_module.WhatsAppAPI
}{apis: make(map[uuid.UUID]*bootstrap_module.WhatsAppAPI)}

// GetWhatsAppAPIByPhoneConfigID returns a WhatsApp API instance for the given phone config ID.
// It caches the API instance for reuse.
func GetWhatsAppAPIByPhoneConfigID(phoneConfigID uuid.UUID) (*bootstrap_module.WhatsAppAPI, error) {
	// Check cache first
	apiCache.RLock()
	if api, ok := apiCache.apis[phoneConfigID]; ok {
		apiCache.RUnlock()
		return api, nil
	}
	apiCache.RUnlock()

	// Load phone config from database
	var phoneConfig phone_config_entity.PhoneConfig
	if err := database.DB.First(&phoneConfig, phoneConfigID).Error; err != nil {
		return nil, fmt.Errorf("phone config not found: %w", err)
	}

	return buildWhatsAppAPI(&phoneConfig)
}

// GetWhatsAppAPIByWabaID returns a WhatsApp API instance for the given WABA ID (Phone Number ID).
func GetWhatsAppAPIByWabaID(wabaID string) (*bootstrap_module.WhatsAppAPI, *phone_config_entity.PhoneConfig, error) {
	var phoneConfig phone_config_entity.PhoneConfig
	if err := database.DB.Where("waba_id = ? AND is_active = true", wabaID).First(&phoneConfig).Error; err != nil {
		return nil, nil, fmt.Errorf("phone config not found for waba_id %s: %w", wabaID, err)
	}

	api, err := buildWhatsAppAPI(&phoneConfig)
	if err != nil {
		return nil, nil, err
	}

	return api, &phoneConfig, nil
}

// buildWhatsAppAPI creates and caches a WhatsApp API instance from a phone config.
func buildWhatsAppAPI(phoneConfig *phone_config_entity.PhoneConfig) (*bootstrap_module.WhatsAppAPI, error) {
	version := "v24.0"
	wabaApi, err := bootstrap_service.GenerateWhatsAppAPI(phoneConfig.AccessToken, &version, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate WhatsApp API: %w", err)
	}

	if _, err := wabaApi.SetWABAID(phoneConfig.WabaID); err != nil {
		return nil, fmt.Errorf("failed to set WABA ID: %w", err)
	}

	if _, err := wabaApi.SetWABAAccountID(phoneConfig.WabaAccountID); err != nil {
		return nil, fmt.Errorf("failed to set WABA account ID: %w", err)
	}

	wabaApi.SetJSONHeaders().SetFormHeaders().SetWABAIDURL(nil)
	wabaApi.SetWABAAccountIDURL(nil)

	// Cache the API instance
	apiCache.Lock()
	apiCache.apis[phoneConfig.ID] = wabaApi
	apiCache.Unlock()

	return wabaApi, nil
}

// InvalidateCache removes a phone config from the cache.
// Call this when a phone config is updated or deleted.
func InvalidateCache(phoneConfigID uuid.UUID) {
	apiCache.Lock()
	delete(apiCache.apis, phoneConfigID)
	apiCache.Unlock()
}

// ClearCache removes all cached API instances.
func ClearCache() {
	apiCache.Lock()
	apiCache.apis = make(map[uuid.UUID]*bootstrap_module.WhatsAppAPI)
	apiCache.Unlock()
}

// GetMetaAppSecret returns the Meta app secret for signature verification.
func GetMetaAppSecret(phoneConfig *phone_config_entity.PhoneConfig) string {
	return phoneConfig.MetaAppSecret
}

// GetVerifyToken returns the webhook verify token.
func GetVerifyToken(phoneConfig *phone_config_entity.PhoneConfig) string {
	return phoneConfig.WebhookVerifyToken
}
