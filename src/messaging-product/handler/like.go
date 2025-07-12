package messaging_product_handler

import (
	"net/url"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	"github.com/Astervia/wacraft-server/src/database"
	messaging_product_service "github.com/Astervia/wacraft-server/src/messaging-product/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// ContactContentLike returns a paginated list of messaging product contacts matching a text pattern.
//
//	@Summary		Search contacts by content
//	@Description	Returns a paginated list of messaging product contacts where the provided text matches contact name, email, or product_details fields using regex (~).
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			paginate	query		messaging_product_model.QueryContactPaginated		true	"Query and pagination parameters"
//	@Param			likeText	path		string												true	"Text to apply like (~) operator"
//	@Success		200			{array}		messaging_product_entity.MessagingProductContact	"List of matching contacts"
//	@Failure		400			{object}	common_model.DescriptiveError						"Invalid query or likeText"
//	@Failure		500			{object}	common_model.DescriptiveError						"Failed to retrieve contacts"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact/content/like/{likeText} [get]
func ContactContentLike(c *fiber.Ctx) error {
	encodedText := c.Params("likeText")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode likeText", err, "net/url").Send(),
		)
	}

	var query messaging_product_model.QueryContactPaginated
	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
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

	mps, err := messaging_product_service.ContactContentLike(
		decodedText,
		mpc,
		&query.Paginate,
		&query.DateOrder,
		&query.DateWhere,
		db,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to get messaging products contacts", err, "messaging_product_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(mps)
}
