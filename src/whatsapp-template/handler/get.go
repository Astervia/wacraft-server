package whatsapp_template_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	phone_config_service "github.com/Astervia/wacraft-server/src/phone-config/service"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	template "github.com/Rfluid/whatsapp-cloud-api/src/template"
	template_model "github.com/Rfluid/whatsapp-cloud-api/src/template"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Get retrieves WhatsApp templates from the Graph API with pagination.
//
//	@Summary		Get WhatsApp templates
//	@Description	Retrieves a paginated list of WhatsApp templates using the Graph API.
//	@Tags			WhatsApp template
//	@Accept			json
//	@Produce		json
//	@Param			template	query		template_model.TemplateQueryParams	true	"Pagination and query parameters"
//	@Success		200			{array}		template_model.GetTemplateResponse	"List of templates"
//	@Failure		400			{object}	common_model.DescriptiveError		"Invalid query parameters"
//	@Failure		500			{object}	common_model.DescriptiveError		"Unable to retrieve templates from API"
//	@Router			/whatsapp-template [get]
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
func Get(c *fiber.Ctx) error {
	query := new(template_model.TemplateQueryParams)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewParseJsonError(err).Send())
	}
	if err := validators.Validator().Struct(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewValidationError(err).Send())
	}

	workspace := workspace_middleware.GetWorkspace(c)
	api, err := phone_config_service.GetFirstConfigAPIByWorkspace(workspace.ID)
	if err == gorm.ErrRecordNotFound {
		return c.Status(fiber.StatusOK).JSON(template.GetTemplateResponse{})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get WhatsApp API", err, "phone_config_service").Send(),
		)
	} else if api == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError(
				"empty WhatsApp API", nil, "phone_config_service",
			).Send(),
		)
	}

	templates, err := template.Get(*api, *query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get templates", err, "template_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(templates)
}
