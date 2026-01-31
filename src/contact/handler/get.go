package contact_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	contact_model "github.com/Astervia/wacraft-core/src/contact/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// Get returns a paginated list of contacts.
//
//	@Summary		Get contacts paginated
//	@Description	Returns a paginated list of contacts using optional query parameters for filtering and sorting.
//	@Tags			Contact
//	@Accept			json
//	@Produce		json
//	@Param			paginate	query		contact_model.QueryPaginated	true	"Query parameters"
//	@Success		200			{array}		contact_entity.Contact			"List of contacts"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/contact [get]
func Get(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	query := new(contact_model.QueryPaginated)
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

	contacts, err := repository.GetPaginated(
		contact_entity.Contact{
			Audit:       common_model.Audit{ID: query.ID},
			Name:        query.Name,
			Email:       query.Email,
			WorkspaceID: &workspace.ID,
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get contacts", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(contacts)
}
