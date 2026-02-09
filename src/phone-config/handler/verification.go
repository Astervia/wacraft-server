package phone_config_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	"github.com/Astervia/wacraft-server/src/database"
	phone_config_service "github.com/Astervia/wacraft-server/src/phone-config/service"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	phone_module "github.com/Rfluid/whatsapp-cloud-api/src/phone"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestCode requests a phone verification code via SMS or voice call.
//
//	@Summary		Request phone verification code
//	@Description	Requests a verification code for the phone number associated with the phone config. The phone config does not need to be active.
//	@Tags			Phone Config
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Param			id				path		string							true	"Phone Config ID"
//	@Param			body			body		phone_module.RequestCodePayload	true	"Request code data"
//	@Success		200				{object}	common.SuccessResponse			"Success"
//	@Failure		400				{object}	common_model.DescriptiveError	"Invalid request"
//	@Failure		404				{object}	common_model.DescriptiveError	"Not found"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace/{workspace_id}/phone-config/{id}/request-code [post]
func RequestCode(c *fiber.Ctx) error {
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

	var req phone_module.RequestCodePayload
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

	result, err := phone_module.RequestCode(*api, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Failed to request verification code", err, "whatsapp").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// VerifyCode verifies a phone verification code.
//
//	@Summary		Verify phone verification code
//	@Description	Verifies the code received on the phone number associated with the phone config. The phone config does not need to be active.
//	@Tags			Phone Config
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Param			id				path		string							true	"Phone Config ID"
//	@Param			body			body		phone_module.VerifyCodePayload	true	"Verify code data"
//	@Success		200				{object}	common.SuccessResponse			"Success"
//	@Failure		400				{object}	common_model.DescriptiveError	"Invalid request"
//	@Failure		404				{object}	common_model.DescriptiveError	"Not found"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace/{workspace_id}/phone-config/{id}/verify-code [post]
func VerifyCode(c *fiber.Ctx) error {
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

	var req phone_module.VerifyCodePayload
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

	result, err := phone_module.VerifyCode(*api, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Failed to verify code", err, "whatsapp").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}
