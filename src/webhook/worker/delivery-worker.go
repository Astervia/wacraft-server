package webhook_worker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_core_service "github.com/Astervia/wacraft-core/src/webhook/service"
	"github.com/Astervia/wacraft-server/src/database"
	webhook_service "github.com/Astervia/wacraft-server/src/webhook/service"
	"github.com/pterm/pterm"
	"golang.org/x/sync/errgroup"
)

const (
	// PollInterval is how often the worker checks for pending deliveries
	PollInterval = 5 * time.Second
	// PoolSize is the max number of concurrent delivery workers
	PoolSize = 10
	// BatchSize is the max number of deliveries to fetch per poll
	BatchSize = 50
)

// DeliveryWorker handles webhook delivery processing
type DeliveryWorker struct {
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	httpClient *http.Client
}

// NewDeliveryWorker creates a new delivery worker
func NewDeliveryWorker() *DeliveryWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &DeliveryWorker{
		ctx:    ctx,
		cancel: cancel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Default timeout, will be overridden per-request
		},
	}
}

// Start begins the delivery worker
func (w *DeliveryWorker) Start() {
	w.wg.Add(1)
	go w.run()
	pterm.DefaultLogger.Info("Webhook delivery worker started")
}

// Stop gracefully stops the delivery worker
func (w *DeliveryWorker) Stop() {
	pterm.DefaultLogger.Info("Stopping webhook delivery worker...")
	w.cancel()
	w.wg.Wait()
	pterm.DefaultLogger.Info("Webhook delivery worker stopped")
}

// run is the main loop that polls for pending deliveries
func (w *DeliveryWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.processPendingDeliveries()
		}
	}
}

// processPendingDeliveries fetches and processes pending deliveries
func (w *DeliveryWorker) processPendingDeliveries() {
	deliveries, err := webhook_service.GetPendingDeliveries(BatchSize)
	if err != nil {
		pterm.DefaultLogger.Error("Failed to get pending deliveries: " + err.Error())
		return
	}

	if len(deliveries) == 0 {
		return
	}

	// Use errgroup with limited concurrency
	g, ctx := errgroup.WithContext(w.ctx)
	g.SetLimit(PoolSize)

	for i := range deliveries {
		delivery := &deliveries[i] // Create a copy for the goroutine
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				w.processDelivery(delivery)
				return nil
			}
		})
	}

	if err := g.Wait(); err != nil && err != context.Canceled {
		pterm.DefaultLogger.Error("Error processing deliveries: " + err.Error())
	}
}

// processDelivery handles a single delivery attempt
func (w *DeliveryWorker) processDelivery(delivery *webhook_entity.WebhookDelivery) {
	if delivery.Webhook == nil {
		// Load webhook if not preloaded
		var webhook webhook_entity.Webhook
		if err := database.DB.First(&webhook, "id = ?", delivery.WebhookID).Error; err != nil {
			pterm.DefaultLogger.Error("Failed to load webhook: " + err.Error())
			errMsg := err.Error()
			webhook_service.UpdateDeliveryStatus(delivery, false, 0, "", errMsg)
			return
		}
		delivery.Webhook = &webhook
	}

	// Check circuit breaker
	cb := webhook_core_service.NewCircuitBreaker(database.DB)
	allowed, err := cb.AllowRequest(delivery.WebhookID)
	if err != nil {
		pterm.DefaultLogger.Error("Circuit breaker check failed: " + err.Error())
		return // Don't update status, will retry later
	}
	if !allowed {
		pterm.DefaultLogger.Warn("Circuit open for webhook: " + delivery.WebhookID.String())
		return // Don't update status, will retry when circuit closes
	}

	// Execute the webhook
	startTime := time.Now()
	httpCode, responseBody, err := w.executeWebhook(delivery)
	duration := time.Since(startTime)

	// Determine success (2xx status codes)
	success := err == nil && httpCode >= 200 && httpCode < 300

	// Create log entry
	w.createLogEntry(delivery, httpCode, responseBody, err, duration, success)

	// Update circuit breaker
	if success {
		cb.RecordSuccess(delivery.WebhookID)
	} else {
		cb.RecordFailure(delivery.WebhookID)
	}

	// Update delivery status
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	if updateErr := webhook_service.UpdateDeliveryStatus(delivery, success, httpCode, responseBody, errMsg); updateErr != nil {
		pterm.DefaultLogger.Error("Failed to update delivery status: " + updateErr.Error())
	}
}

// executeWebhook makes the HTTP request to the webhook endpoint
func (w *DeliveryWorker) executeWebhook(delivery *webhook_entity.WebhookDelivery) (int, string, error) {
	webhook := delivery.Webhook

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(delivery.Payload)
	if err != nil {
		return 0, "", err
	}

	// Create request
	req, err := http.NewRequest(webhook.HttpMethod, webhook.Url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return 0, "", err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Wacraft-Delivery-ID", delivery.ID.String())
	req.Header.Set("X-Wacraft-Event", delivery.EventType)
	req.Header.Set("X-Wacraft-Attempt", strconv.Itoa(delivery.AttemptCount+1))

	// Add authorization header if set
	if webhook.Authorization != "" {
		req.Header.Set("Authorization", webhook.Authorization)
	}

	// Add custom headers
	for key, value := range webhook.CustomHeaders {
		req.Header.Set(key, value)
	}

	// Add signature headers if enabled
	if webhook.SigningEnabled && webhook.SigningSecret != "" {
		signature, timestamp := webhook_core_service.SignatureHeaders(webhook.SigningSecret, jsonPayload)
		req.Header.Set("X-Wacraft-Signature", signature)
		req.Header.Set("X-Wacraft-Timestamp", timestamp)
	}

	// Set timeout
	timeout := 30 * time.Second // Default
	if webhook.Timeout != nil && *webhook.Timeout > 0 {
		timeout = time.Duration(*webhook.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// Execute request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	// Read response body (limited to prevent memory issues)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024)) // 64KB limit
	if err != nil {
		return resp.StatusCode, "", err
	}

	return resp.StatusCode, string(bodyBytes), nil
}

// createLogEntry creates a webhook log entry for the attempt
func (w *DeliveryWorker) createLogEntry(delivery *webhook_entity.WebhookDelivery, httpCode int, responseBody string, execErr error, duration time.Duration, success bool) {
	// Parse response data if JSON
	var responseData any
	if responseBody != "" {
		if err := json.Unmarshal([]byte(responseBody), &responseData); err != nil {
			responseData = nil
		}
	}

	// Build request headers map
	requestHeaders := map[string]string{
		"Content-Type":          "application/json",
		"X-Wacraft-Delivery-ID": delivery.ID.String(),
		"X-Wacraft-Event":       delivery.EventType,
		"X-Wacraft-Attempt":     strconv.Itoa(delivery.AttemptCount + 1),
	}
	if delivery.Webhook.Authorization != "" {
		requestHeaders["Authorization"] = "[REDACTED]"
	}
	for key := range delivery.Webhook.CustomHeaders {
		requestHeaders[key] = delivery.Webhook.CustomHeaders[key]
	}
	signatureSent := delivery.Webhook.SigningEnabled && delivery.Webhook.SigningSecret != ""
	if signatureSent {
		requestHeaders["X-Wacraft-Signature"] = "[REDACTED]"
		requestHeaders["X-Wacraft-Timestamp"] = "[SET]"
	}

	log := webhook_entity.WebhookLog{
		Payload:          delivery.Payload,
		HttpResponseCode: httpCode,
		ResponseData:     responseData,
		WebhookID:        delivery.WebhookID,
		DeliveryID:       &delivery.ID,
		AttemptNumber:    delivery.AttemptCount + 1,
		DurationMs:       duration.Milliseconds(),
		SignatureSent:    signatureSent,
		IdempotencyKey:   delivery.IdempotencyKey,
		RequestHeaders:   requestHeaders,
		RequestUrl:       delivery.Webhook.Url,
	}

	if err := database.DB.Create(&log).Error; err != nil {
		pterm.DefaultLogger.Error("Failed to create webhook log: " + err.Error())
	}
}
