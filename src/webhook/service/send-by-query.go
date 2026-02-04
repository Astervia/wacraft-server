package webhook_service

import (
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/pterm/pterm"
)

// SendAllByQuery enqueues deliveries for all webhooks matching the query.
// This replaces the old fire-and-forget approach with queue-based delivery
// that supports retries, signatures, and circuit breakers.
func SendAllByQuery(
	entity webhook_entity.Webhook,
	payload interface{},
) error {
	var webhooks []webhook_entity.Webhook

	// Query active webhooks matching the criteria
	if err := database.DB.
		Where(&entity).
		Where("is_active = ?", true).
		Find(&webhooks).Error; err != nil {
		return err
	}

	// Enqueue delivery for each webhook
	for i := range webhooks {
		if err := EnqueueDelivery(&webhooks[i], payload, string(entity.Event)); err != nil {
			// Log error but continue processing other webhooks
			pterm.DefaultLogger.Error("Failed to enqueue delivery for webhook " + webhooks[i].ID.String() + ": " + err.Error())
		}
	}

	return nil
}
