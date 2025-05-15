package messaging_product_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// CreateContact creates a new contact for a messaging product.
//	@Summary		Creates a new messaging product contact
//	@Description	Creates and stores a new contact associated with a messaging product.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			contact	body		messaging_product_model.CreateContact				true	"Contact data"
//	@Success		201		{object}	messaging_product_entity.MessagingProductContact	"Created contact"
//	@Failure		400		{object}	common_model.DescriptiveError						"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError						"Failed to create contact"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact [post]
func CreateContact(c *fiber.Ctx) error {
	// Parse the request body
	var data messaging_product_model.CreateContact
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	// Create the new user
	newEntity := messaging_product_entity.MessagingProductContact{
		ContactId:          data.ContactId,
		MessagingProductId: data.MessagingProductId,
		ProductDetails:     &data.ProductDetails,
	}

	// Save the new user to the database
	if err := database.DB.Create(&newEntity).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create messaging product contact", err, "gorm.io/gorm").Send(),
		)
	}

	// Return the created user (or just a success message)
	return c.Status(fiber.StatusCreated).JSON(newEntity)
}
