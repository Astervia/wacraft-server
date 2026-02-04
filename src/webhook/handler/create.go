package webhook_handler

import (
	"crypto/rand"
	"encoding/hex"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	webhook_model "github.com/Astervia/wacraft-core/src/webhook/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// CreateWebhook registers a new webhook for event notifications.
//
//	@Summary		Create a new webhook
//	@Description	Creates a new webhook with the specified URL, authorization header, method, timeout, and event type.
//	@Tags			Webhook
//	@Accept			json
//	@Produce		json
//	@Param			webhook	body		webhook_model.CreateWebhook		true	"Webhook data"
//	@Success		201		{object}	webhook_entity.Webhook			"Created webhook"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/webhook [post]
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
func CreateWebhook(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var newWebhook webhook_model.CreateWebhook
	if err := c.BodyParser(&newWebhook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&newWebhook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	// Build webhook entity
	webhookEntity := webhook_entity.Webhook{
		Url:            newWebhook.Url,
		Authorization:  newWebhook.Authorization,
		HttpMethod:     newWebhook.HttpMethod,
		Timeout:        newWebhook.Timeout,
		Event:          newWebhook.Event,
		WorkspaceID:    &workspace.ID,
		SigningEnabled: newWebhook.SigningEnabled,
		CustomHeaders:  newWebhook.CustomHeaders,
		EventFilter:    newWebhook.EventFilter,
		IsActive:       true, // Default to active
	}

	// Set max retries if provided
	if newWebhook.MaxRetries != nil {
		webhookEntity.MaxRetries = *newWebhook.MaxRetries
	} else {
		webhookEntity.MaxRetries = 3 // Default
	}

	// Set retry delay if provided
	if newWebhook.RetryDelayMs != nil {
		webhookEntity.RetryDelayMs = *newWebhook.RetryDelayMs
	} else {
		webhookEntity.RetryDelayMs = 1000 // Default 1 second
	}

	// Generate signing secret if signing is enabled
	var signingSecret string
	if newWebhook.SigningEnabled {
		secret, err := generateSigningSecret()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("unable to generate signing secret", err, "crypto").Send(),
			)
		}
		webhookEntity.SigningSecret = secret
		signingSecret = secret // Save to return in response
	}

	webhook, err := repository.Create(webhookEntity, database.DB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create webhook", err, "repository").Send(),
		)
	}

	// Build response - include signing secret only on creation
	response := CreateWebhookResponse{
		Webhook:       webhook,
		SigningSecret: signingSecret,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// CreateWebhookResponse is the response for webhook creation
// It includes the signing secret which is only returned once
type CreateWebhookResponse struct {
	webhook_entity.Webhook
	SigningSecret string `json:"signing_secret,omitempty"` // Only returned on creation when signing is enabled
}

// generateSigningSecret generates a cryptographically secure random secret
func generateSigningSecret() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(bytes), nil
}
