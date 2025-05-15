package contact_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	contact_model "github.com/Astervia/wacraft-core/src/contact/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// Get returns a paginated list of contacts.
//	@Summary		Get contacts paginated
//	@Description	Returns a paginated list of contacts.
//	@Tags			Contact
//	@Accept			json
//	@Produce		json
//	@Param			paginate	query		contact_model.QueryPaginated	true	"Query parameters"
//	@Success		200			{array}		contact_entity.Contact			"List of contacts"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid query parameters"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/contact [get]
func Get(c *fiber.Ctx) error {
	query := new(contact_model.QueryPaginated)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	contacts, err := repository.GetPaginated(
		contact_entity.Contact{
			Audit: common_model.Audit{Id: query.Id},
			Name:  query.Name,
			Email: query.Email,
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
