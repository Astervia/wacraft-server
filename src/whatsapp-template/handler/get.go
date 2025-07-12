package whatsapp_template_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-server/src/integration/whatsapp"
	"github.com/Astervia/wacraft-server/src/validators"
	template_model "github.com/Rfluid/whatsapp-cloud-api/src/template/model"
	template_service "github.com/Rfluid/whatsapp-cloud-api/src/template/service"
	"github.com/gofiber/fiber/v2"
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
func Get(c *fiber.Ctx) error {
	query := new(template_model.TemplateQueryParams)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewParseJsonError(err).Send())
	}

	if err := validators.Validator().Struct(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewValidationError(err).Send())
	}

	template, err := template_service.Get(whatsapp.WabaApi, *query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get templates", err, "template_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(template)
}
