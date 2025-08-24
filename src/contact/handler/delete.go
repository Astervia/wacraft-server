package contact_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// DeleteContactByID deletes a contact using the provided ID.
//
//	@Summary		Delete contact by ID
//	@Description	Deletes a contact based on the ID sent in the request body.
//	@Tags			Contact
//	@Accept			json
//	@Produce		json
//	@Param			body	body		common_model.RequiredID			true	"Contact ID to delete"
//	@Success		204		{string}	string							"Contact deleted successfully"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/contact [delete]
func DeleteContactByID(c *fiber.Ctx) error {
	var reqBody common_model.RequiredID
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

	err := repository.DeleteByID[contact_entity.Contact](reqBody.ID, database.DB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to delete contact", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
