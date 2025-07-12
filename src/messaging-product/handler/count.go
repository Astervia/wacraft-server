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

// ContactContentLikeCount returns the number of messaging product contacts matching a text pattern.
//
//	@Summary		Count contacts by content
//	@Description	Returns the number of messaging product contacts that match the provided text (regex ~) in fields like name, email, and product_details.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			paginate	query		messaging_product_model.QueryContact	true	"Query parameters"
//	@Param			likeText	path		string									true	"Text to match using regex (~)"
//	@Success		200			{integer}	int										"Count of matching contacts"
//	@Failure		400			{object}	common_model.DescriptiveError			"Invalid query or likeText"
//	@Failure		500			{object}	common_model.DescriptiveError			"Failed to count contacts"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact/count/content/like/{likeText} [get]
func ContactContentLikeCount(c *fiber.Ctx) error {
	encodedText := c.Params("likeText")
	decodedText, err := url.QueryUnescape(encodedText)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to decode likeText", err, "net/url").Send(),
		)
	}

	var query messaging_product_model.QueryContact
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

	mps, err := messaging_product_service.ContactContentLikeCount(
		decodedText,
		mpc,
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
