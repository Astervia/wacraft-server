package messaging_product_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	_ "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// GetContact returns a paginated list of messaging product contacts.
//	@Summary		Get messaging products contacts paginated
//	@Description	Returns a paginated list of messaging product contacts, joining with the contact entity.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			paginate	query		messaging_product_model.QueryContactPaginated		true	"Query and pagination parameters"
//	@Success		200			{array}		messaging_product_entity.MessagingProductContact	"List of messaging product contacts"
//	@Failure		400			{object}	common_model.DescriptiveError						"Invalid query parameters"
//	@Failure		500			{object}	common_model.DescriptiveError						"Failed to retrieve contacts"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact [get]
func GetContact(c *fiber.Ctx) error {
	var query messaging_product_model.QueryContactPaginated
	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	mpc := messaging_product_entity.MessagingProductContact{
		ContactId:          query.ContactID,
		MessagingProductId: query.MessagingProductID,
		Audit: common_model.Audit{
			Id: query.Id,
		},
	}

	db := database.DB.Model(&mpc)

	if mpc.ProductDetails != nil {
		mpc.ProductDetails.ParseIndividualFieldQueries(&db)
		mpc.ProductDetails = nil
	}
	db = db.Joins("Contact")

	mps, err := repository.GetPaginated(
		mpc,
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"",
		db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get messaging product contacts", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(mps)
}

// GetWhatsAppContact returns a paginated list of WhatsApp messaging product contacts.
//	@Summary		Get WhatsApp messaging products contacts paginated
//	@Description	Queries a paginated list of WhatsApp messaging product contacts, including WhatsApp-specific fields and joining with the contact entity.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			paginate	query		messaging_product_model.QueryWhatsAppContactPaginated	true	"Query and pagination parameters"
//	@Success		200			{array}		messaging_product_entity.MessagingProductContact		"List of WhatsApp messaging product contacts"
//	@Failure		400			{object}	common_model.DescriptiveError							"Invalid query parameters"
//	@Failure		500			{object}	common_model.DescriptiveError							"Failed to retrieve WhatsApp contacts"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact/whatsapp [get]
func GetWhatsAppContact(c *fiber.Ctx) error {
	query := new(messaging_product_model.QueryWhatsAppContactPaginated)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err})
	}

	mpc := messaging_product_entity.MessagingProductContact{
		ContactId: query.ContactID,
		ProductDetails: &messaging_product_model.ProductDetails{
			WhatsAppProductDetails: &messaging_product_model.WhatsAppProductDetails{
				PhoneNumber: query.PhoneNumber,
				WaId:        query.WaId,
			},
		},
		MessagingProduct: &messaging_product_entity.MessagingProduct{
			Name: messaging_product_model.WhatsApp,
		},
		Audit: common_model.Audit{
			Id: query.Id,
		},
	}

	db := database.DB.Model(&mpc)

	if mpc.ProductDetails != nil {
		mpc.ProductDetails.ParseIndividualFieldQueries(&db)
		mpc.ProductDetails = nil
	}
	db = db.Joins("Contact")

	mps, err := repository.GetPaginated(
		mpc,
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		"",
		db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err})
	}

	return c.Status(fiber.StatusOK).JSON(mps)
}
