package phone_config_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	phone_config_model "github.com/Astervia/wacraft-core/src/phone-config/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	phone_config_service "github.com/Astervia/wacraft-server/src/phone-config/service"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	bootstrap_service "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap/service"
	profile_service "github.com/Rfluid/whatsapp-cloud-api/src/profile/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// validatePhoneConfigOwnership verifies that the provided credentials can access the phone number.
// This prevents users from registering phone numbers they don't own.
func validatePhoneConfigOwnership(wabaID string, accessToken string) error {
	version := "v24.0"
	api, err := bootstrap_service.GenerateWhatsAppAPI(accessToken, &version, nil)
	if err != nil {
		return err
	}

	if _, err := api.SetWABAID(wabaID); err != nil {
		return err
	}

	api.SetWABAIDURL(nil)
	api.SetJSONHeaders()

	// Attempt to get the profile to verify ownership
	// We only request the 'about' field as a minimal check
	_, err = profile_service.GetProfile(*api, nil)

	return err
}

// deactivateConflictingPhoneConfigs deactivates all active phone configs with the same WabaID.
func deactivateConflictingPhoneConfigs(wabaID string) error {
	return database.DB.Model(&phone_config_entity.PhoneConfig{}).
		Where("waba_id = ? AND is_active = true", wabaID).
		Update("is_active", false).Error
}

// Create creates a new phone configuration.
//
//	@Summary		Create phone configuration
//	@Description	Creates a new phone configuration for the workspace.
//	@Tags			Phone Config
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Param			config			body		phone_config_model.CreatePhoneConfig	true	"Phone config data"
//	@Success		201				{object}	phone_config_entity.PhoneConfig	"Created phone config"
//	@Failure		400				{object}	common_model.DescriptiveError	"Invalid request"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/workspace/{workspace_id}/phone-config [post]
func Create(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var req phone_config_model.CreatePhoneConfig
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

	// Validate ownership by attempting to get the profile
	if err := validatePhoneConfigOwnership(req.WabaID, req.AccessToken); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError(err.Error(), err, "validation").Send(),
		)
	}

	// Deactivate any existing active phone configs with the same WabaID
	if err := deactivateConflictingPhoneConfigs(req.WabaID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("failed to deactivate conflicting phone configs", err, "database").Send(),
		)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	phoneConfig, err := repository.Create(
		phone_config_entity.PhoneConfig{
			Name:               req.Name,
			WorkspaceID:        &workspace.ID,
			WabaID:             req.WabaID,
			WabaAccountID:      req.WabaAccountID,
			DisplayPhone:       req.DisplayPhone,
			AccessToken:        req.AccessToken,
			MetaAppSecret:      req.MetaAppSecret,
			WebhookVerifyToken: req.WebhookVerifyToken,
			IsActive:           isActive,
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to create phone config", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(phoneConfig)
}

// Get returns a paginated list of phone configurations.
//
//	@Summary		List phone configurations
//	@Description	Returns a paginated list of phone configurations for the workspace.
//	@Tags			Phone Config
//	@Produce		json
//	@Param			workspace_id	path		string								true	"Workspace ID"
//	@Param			paginate		query		phone_config_model.QueryPaginated	true	"Query parameters"
//	@Success		200				{array}		phone_config_entity.PhoneConfig		"List of phone configs"
//	@Failure		400				{object}	common_model.DescriptiveError		"Invalid request"
//	@Failure		500				{object}	common_model.DescriptiveError		"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/workspace/{workspace_id}/phone-config [get]
func Get(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	query := new(phone_config_model.QueryPaginated)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	phoneConfigs, err := repository.GetPaginated(
		phone_config_entity.PhoneConfig{
			Audit:        common_model.Audit{ID: query.ID},
			WorkspaceID:  &workspace.ID,
			Name:         query.Name,
			WabaID:       query.WabaID,
			DisplayPhone: query.DisplayPhone,
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("failed to get phone configs", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(phoneConfigs)
}

// GetByID returns a specific phone configuration.
//
//	@Summary		Get phone configuration
//	@Description	Returns a specific phone configuration by ID.
//	@Tags			Phone Config
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Param			id				path		string							true	"Phone Config ID"
//	@Success		200				{object}	phone_config_entity.PhoneConfig	"Phone config"
//	@Failure		404				{object}	common_model.DescriptiveError	"Not found"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/workspace/{workspace_id}/phone-config/{id} [get]
func GetByID(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid phone config ID", err, "handler").Send(),
		)
	}

	var phoneConfig phone_config_entity.PhoneConfig
	if err := database.DB.Where("id = ? AND workspace_id = ?", id, workspace.ID).First(&phoneConfig).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Phone config not found", err, "database").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(phoneConfig)
}

// Update updates an existing phone configuration.
//
//	@Summary		Update phone configuration
//	@Description	Updates an existing phone configuration.
//	@Tags			Phone Config
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Param			id				path		string							true	"Phone Config ID"
//	@Param			config			body		phone_config_model.UpdatePhoneConfig	true	"Phone config data"
//	@Success		200				{object}	phone_config_entity.PhoneConfig	"Updated phone config"
//	@Failure		400				{object}	common_model.DescriptiveError	"Invalid request"
//	@Failure		404				{object}	common_model.DescriptiveError	"Not found"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/workspace/{workspace_id}/phone-config/{id} [patch]
func Update(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid phone config ID", err, "handler").Send(),
		)
	}

	var req phone_config_model.UpdatePhoneConfig
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	// Find existing config
	var phoneConfig phone_config_entity.PhoneConfig
	if err := database.DB.Where("id = ? AND workspace_id = ?", id, workspace.ID).First(&phoneConfig).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Phone config not found", err, "database").Send(),
		)
	}

	// Validate ownership if WabaID, AccessToken, or IsActive is being changed
	if req.WabaID != nil ||
		req.AccessToken != nil ||
		(req.IsActive != nil && *req.IsActive == true) {
		wabaID := phoneConfig.WabaID
		if req.WabaID != nil {
			wabaID = *req.WabaID
		}

		var accessToken string
		if req.AccessToken != nil {
			accessToken = *req.AccessToken
		} else {
			accessToken = phoneConfig.AccessToken
		}

		if err := validatePhoneConfigOwnership(wabaID, accessToken); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				common_model.NewApiError(err.Error(), err, "validation").Send(),
			)
		}

		// Deactivate conflicting phone configs if activating
		if req.IsActive != nil && *req.IsActive {
			if err := deactivateConflictingPhoneConfigs(wabaID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(
					common_model.NewApiError("Failed to deactivate conflicting phone configs", err, "database").Send(),
				)
			}
		}
	}

	// Build updates
	updates := make(map[string]any)

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.WabaID != nil {
		updates["waba_id"] = *req.WabaID
	}
	if req.WabaAccountID != nil {
		updates["waba_account_id"] = *req.WabaAccountID
	}
	if req.DisplayPhone != nil {
		updates["display_phone"] = *req.DisplayPhone
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.AccessToken != nil {
		updates["access_token"] = *req.AccessToken
	}
	if req.MetaAppSecret != nil {
		updates["meta_app_secret"] = *req.MetaAppSecret
	}
	if req.WebhookVerifyToken != nil {
		updates["webhook_verify_token"] = *req.WebhookVerifyToken
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&phoneConfig).Updates(updates).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("Failed to update phone config", err, "database").Send(),
			)
		}
		// Invalidate cached API instance
		phone_config_service.InvalidateCache(id)
	}

	// Reload to get updated values
	if err := database.DB.First(&phoneConfig, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to reload phone config", err, "database").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(phoneConfig)
}

// Delete deletes a phone configuration.
//
//	@Summary		Delete phone configuration
//	@Description	Deletes a phone configuration from the workspace.
//	@Tags			Phone Config
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Param			id				path		string							true	"Phone Config ID"
//	@Success		204				{object}	nil								"Deleted"
//	@Failure		404				{object}	common_model.DescriptiveError	"Not found"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/workspace/{workspace_id}/phone-config/{id} [delete]
func Delete(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid phone config ID", err, "handler").Send(),
		)
	}

	result := database.DB.Where("id = ? AND workspace_id = ?", id, workspace.ID).Delete(&phone_config_entity.PhoneConfig{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to delete phone config", result.Error, "database").Send(),
		)
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Phone config not found", nil, "handler").Send(),
		)
	}

	// Invalidate cached API instance
	phone_config_service.InvalidateCache(id)

	return c.SendStatus(fiber.StatusNoContent)
}
