package webhook_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_core_service "github.com/Astervia/wacraft-core/src/webhook/service"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// TestWebhookRequest represents the request body for testing a webhook
type TestWebhookRequest struct {
	WebhookID uuid.UUID `json:"webhook_id" validate:"required"`
	Payload   any       `json:"payload,omitempty"` // Optional custom payload, defaults to test payload
}

// TestWebhookResponse represents the response from testing a webhook
type TestWebhookResponse struct {
	Success      bool              `json:"success"`
	StatusCode   int               `json:"status_code,omitempty"`
	Response     any               `json:"response,omitempty"`
	ResponseBody string            `json:"response_body,omitempty"`
	DurationMs   int64             `json:"duration_ms"`
	HeadersSent  map[string]string `json:"headers_sent"`
	Error        string            `json:"error,omitempty"`
}

// TestWebhook sends a test payload to a webhook and returns the result.
//
//	@Summary		Test a webhook
//	@Description	Sends a test payload to the specified webhook URL and returns the response details.
//	@Tags			Webhook
//	@Accept			json
//	@Produce		json
//	@Param			request	body		TestWebhookRequest				true	"Test request"
//	@Success		200		{object}	TestWebhookResponse				"Test result"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		404		{object}	common_model.DescriptiveError	"Webhook not found"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/webhook/test [post]
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
func TestWebhook(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var req TestWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	// Get the webhook
	var webhook webhook_entity.Webhook
	if err := database.DB.Where("id = ? AND workspace_id = ?", req.WebhookID, workspace.ID).First(&webhook).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("webhook not found", err, "database").Send(),
		)
	}

	// Use default test payload if none provided
	payload := req.Payload
	if payload == nil {
		payload = map[string]any{
			"test":       true,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"webhook_id": webhook.ID.String(),
			"event":      string(webhook.Event),
			"message":    "This is a test webhook delivery from Wacraft",
		}
	}

	// Execute the test
	result := executeTestWebhook(&webhook, payload)

	return c.Status(fiber.StatusOK).JSON(result)
}

func executeTestWebhook(webhook *webhook_entity.Webhook, payload any) TestWebhookResponse {
	result := TestWebhookResponse{
		HeadersSent: make(map[string]string),
	}

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		result.Error = "Failed to marshal payload: " + err.Error()
		return result
	}

	// Create request
	req, err := http.NewRequest(webhook.HttpMethod, webhook.Url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		result.Error = "Failed to create request: " + err.Error()
		return result
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	result.HeadersSent["Content-Type"] = "application/json"

	req.Header.Set("X-Wacraft-Test", "true")
	result.HeadersSent["X-Wacraft-Test"] = "true"

	req.Header.Set("X-Wacraft-Event", string(webhook.Event))
	result.HeadersSent["X-Wacraft-Event"] = string(webhook.Event)

	// Add authorization header if set
	if webhook.Authorization != "" {
		req.Header.Set("Authorization", webhook.Authorization)
		result.HeadersSent["Authorization"] = "[REDACTED]"
	}

	// Add custom headers
	for key, value := range webhook.CustomHeaders {
		req.Header.Set(key, value)
		result.HeadersSent[key] = value
	}

	// Add signature headers if enabled
	if webhook.SigningEnabled && webhook.SigningSecret != "" {
		signature, timestamp := webhook_core_service.SignatureHeaders(webhook.SigningSecret, jsonPayload)
		req.Header.Set("X-Wacraft-Signature", signature)
		req.Header.Set("X-Wacraft-Timestamp", timestamp)
		result.HeadersSent["X-Wacraft-Signature"] = signature
		result.HeadersSent["X-Wacraft-Timestamp"] = timestamp
	}

	// Set timeout
	timeout := 30 * time.Second
	if webhook.Timeout != nil && *webhook.Timeout > 0 {
		timeout = time.Duration(*webhook.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// Execute request
	client := &http.Client{}
	startTime := time.Now()
	resp, err := client.Do(req)
	result.DurationMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Error = "Request failed: " + err.Error()
		result.Success = false
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	// Read response body
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024)) // 64KB limit
	if err != nil {
		result.Error = "Failed to read response: " + err.Error()
		return result
	}

	result.ResponseBody = string(bodyBytes)

	// Try to parse as JSON
	var jsonResponse any
	if err := json.Unmarshal(bodyBytes, &jsonResponse); err == nil {
		result.Response = jsonResponse
	}

	return result
}
