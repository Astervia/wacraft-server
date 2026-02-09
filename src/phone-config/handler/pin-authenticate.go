package phone_config_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	phone_config_service "github.com/Astervia/wacraft-server/src/phone-config/service"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	phone_module "github.com/Rfluid/whatsapp-cloud-api/src/phone"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// PinAuthenticate authenticates the phone number with a two-step verification PIN.
//
//	@Summary		Authenticate with two-step verification PIN
//	@Description	Authenticates the phone number with a two-step verification PIN. The phone config does not need to be active.
//	@Tags			Phone Config
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Param			id				path		string							true	"Phone Config ID"
//	@Param			body			body		phone_module.Pin				true	"PIN data"
//	@Success		200				{object}	common.SuccessResponse			"Success"
//	@Failure		400				{object}	common_model.DescriptiveError	"Invalid request"
//	@Failure		404				{object}	common_model.DescriptiveError	"Not found"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace/{workspace_id}/phone-config/{id}/pin-authenticate [post]
func PinAuthenticate(c *fiber.Ctx) error {
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

	var req phone_module.Pin
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

	api, err := phone_config_service.BuildWhatsAppAPI(&phoneConfig)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to build WhatsApp API", err, "service").Send(),
		)
	}

	result, err := phone_module.AuthenticateWithPin(*api, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Failed to authenticate with PIN", err, "whatsapp").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}
