package messaging_product_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// Get returns a paginated list of messaging products.
//
//	@Summary		Retrieve messaging products
//	@Description	Fetches a paginated list of messaging products using optional filters.
//	@Tags			Messaging product
//	@Accept			json
//	@Produce		json
//	@Param			paginate	query		messaging_product_model.QueryPaginated		true	"Query and pagination parameters"
//	@Success		200			{array}		messaging_product_entity.MessagingProduct	"List of messaging products"
//	@Failure		400			{object}	common_model.DescriptiveError				"Invalid query parameters"
//	@Failure		500			{object}	common_model.DescriptiveError				"Failed to retrieve products"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product [get]
func Get(c *fiber.Ctx) error {
	query := new(messaging_product_model.QueryPaginated)
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

	mps, err := repository.GetPaginated(
		messaging_product_entity.MessagingProduct{
			Name: query.Name,
			Audit: common_model.Audit{
				ID: query.ID,
			},
		},
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"", database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get paginated", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(mps)
}
