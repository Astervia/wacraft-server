package webhook_service

import (
	"encoding/json"
	"fmt"
	"time"

	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_service "github.com/Astervia/wacraft-core/src/webhook/service"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/google/uuid"
)

// EnqueueDelivery creates a new delivery record for a webhook
// It checks the circuit breaker and event filter before enqueuing
func EnqueueDelivery(webhook *webhook_entity.Webhook, payload any, eventType string) error {
	// Check if webhook is active
	if !webhook.IsActive {
		return nil // Silently skip inactive webhooks
	}

	// Check circuit breaker
	cb := webhook_service.NewCircuitBreaker(database.DB)
	allowed, err := cb.AllowRequest(webhook.ID)
	if err != nil {
		return fmt.Errorf("circuit breaker check failed: %w", err)
	}
	if !allowed {
		return nil // Circuit is open, skip this delivery
	}

	// Check event filter
	if !webhook_service.EvaluateFilter(webhook.EventFilter, payload) {
		return nil // Event doesn't match filter, skip
	}

	// Generate idempotency key
	idempotencyKey := generateIdempotencyKey(webhook.ID, eventType, payload)

	// Check for existing delivery with same idempotency key
	var existingCount int64
	if err := database.DB.Model(&webhook_entity.WebhookDelivery{}).
		Where("idempotency_key = ?", idempotencyKey).
		Count(&existingCount).Error; err != nil {
		return fmt.Errorf("failed to check existing delivery: %w", err)
	}
	if existingCount > 0 {
		return nil // Already enqueued
	}

	// Create delivery record
	now := time.Now()
	delivery := webhook_entity.WebhookDelivery{
		WebhookID:      webhook.ID,
		IdempotencyKey: idempotencyKey,
		Payload:        payload,
		Status:         webhook_entity.DeliveryStatusPending,
		AttemptCount:   0,
		MaxAttempts:    webhook.MaxRetries + 1, // Initial attempt + retries
		NextAttemptAt:  &now,
		EventType:      eventType,
		EventTimestamp: now,
	}

	if err := database.DB.Create(&delivery).Error; err != nil {
		return fmt.Errorf("failed to create delivery: %w", err)
	}

	return nil
}

// generateIdempotencyKey creates a unique key for deduplication
func generateIdempotencyKey(webhookID uuid.UUID, eventType string, payload any) string {
	// Use a combination of webhook ID, event type, timestamp, and a random UUID
	// The random UUID ensures uniqueness even for identical payloads at the same timestamp
	return fmt.Sprintf("%s:%s:%d:%s", webhookID.String(), eventType, time.Now().UnixNano(), uuid.New().String())
}

// EnqueueDeliveryWithCustomKey creates a delivery with a custom idempotency key
// This is useful when the caller wants to control deduplication
func EnqueueDeliveryWithCustomKey(webhook *webhook_entity.Webhook, payload any, eventType string, idempotencyKey string) error {
	// Check if webhook is active
	if !webhook.IsActive {
		return nil
	}

	// Check circuit breaker
	cb := webhook_service.NewCircuitBreaker(database.DB)
	allowed, err := cb.AllowRequest(webhook.ID)
	if err != nil {
		return fmt.Errorf("circuit breaker check failed: %w", err)
	}
	if !allowed {
		return nil
	}

	// Check event filter
	if !webhook_service.EvaluateFilter(webhook.EventFilter, payload) {
		return nil
	}

	// Check for existing delivery
	var existingCount int64
	if err := database.DB.Model(&webhook_entity.WebhookDelivery{}).
		Where("idempotency_key = ?", idempotencyKey).
		Count(&existingCount).Error; err != nil {
		return fmt.Errorf("failed to check existing delivery: %w", err)
	}
	if existingCount > 0 {
		return nil
	}

	now := time.Now()
	delivery := webhook_entity.WebhookDelivery{
		WebhookID:      webhook.ID,
		IdempotencyKey: idempotencyKey,
		Payload:        payload,
		Status:         webhook_entity.DeliveryStatusPending,
		AttemptCount:   0,
		MaxAttempts:    webhook.MaxRetries + 1,
		NextAttemptAt:  &now,
		EventType:      eventType,
		EventTimestamp: now,
	}

	if err := database.DB.Create(&delivery).Error; err != nil {
		return fmt.Errorf("failed to create delivery: %w", err)
	}

	return nil
}

// GetPendingDeliveries retrieves deliveries that are ready for processing
func GetPendingDeliveries(limit int) ([]webhook_entity.WebhookDelivery, error) {
	var deliveries []webhook_entity.WebhookDelivery
	now := time.Now()

	err := database.DB.
		Where("status IN ?", []webhook_entity.DeliveryStatus{
			webhook_entity.DeliveryStatusPending,
			webhook_entity.DeliveryStatusAttempted,
		}).
		Where("next_attempt_at <= ?", now).
		Preload("Webhook").
		Order("next_attempt_at ASC").
		Limit(limit).
		Find(&deliveries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get pending deliveries: %w", err)
	}

	return deliveries, nil
}

// UpdateDeliveryStatus updates the status of a delivery after an attempt
func UpdateDeliveryStatus(delivery *webhook_entity.WebhookDelivery, success bool, httpCode int, responseBody string, errMsg string) error {
	now := time.Now()
	delivery.LastAttemptAt = &now
	delivery.AttemptCount++

	if httpCode > 0 {
		delivery.LastHttpCode = &httpCode
	}
	if responseBody != "" {
		delivery.LastResponseBody = &responseBody
	}
	if errMsg != "" {
		delivery.LastError = &errMsg
	}

	if success {
		delivery.Status = webhook_entity.DeliveryStatusSucceeded
		delivery.NextAttemptAt = nil
	} else if delivery.AttemptCount >= delivery.MaxAttempts {
		delivery.Status = webhook_entity.DeliveryStatusDeadLetter
		delivery.NextAttemptAt = nil
	} else {
		delivery.Status = webhook_entity.DeliveryStatusAttempted
		// Exponential backoff: baseDelay * 2^attemptCount (capped at 1 hour)
		baseDelay := time.Duration(delivery.Webhook.RetryDelayMs) * time.Millisecond
		if baseDelay == 0 {
			baseDelay = 1000 * time.Millisecond
		}
		delay := baseDelay * time.Duration(1<<uint(delivery.AttemptCount-1))
		maxDelay := 1 * time.Hour
		if delay > maxDelay {
			delay = maxDelay
		}
		nextAttempt := now.Add(delay)
		delivery.NextAttemptAt = &nextAttempt
	}

	return database.DB.Save(delivery).Error
}

// payloadToJSON converts a payload to JSON bytes
func payloadToJSON(payload any) ([]byte, error) {
	return json.Marshal(payload)
}
