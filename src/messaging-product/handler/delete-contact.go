package messaging_product_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// DeleteContact deletes a messaging product contact by ID.
//
//	@Summary		Delete messaging product contact
//	@Description	Deletes a messaging product contact using the provided ID.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			body	body	common_model.RequiredId	true	"Messaging product contact ID to delete"
//	@Success		204		"Messaging product contact deleted successfully"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Failed to delete contact"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact [delete]
func DeleteContact(c *fiber.Ctx) error {
	var reqBody common_model.RequiredId
	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	err := repository.DeleteById[messaging_product_entity.MessagingProductContact](
		reqBody.Id, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete messaging product contact", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
