package contact_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	contact_entity "github.com/Astervia/wacraft-core/src/contact/entity"
	contact_model "github.com/Astervia/wacraft-core/src/contact/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// CreateContact registers a new contact.
//	@Summary		Create a new contact
//	@Description	Creates a new contact and returns the created object.
//	@Tags			Contact
//	@Accept			json
//	@Produce		json
//	@Param			contact	body		contact_model.CreateContact		true	"Contact data"
//	@Success		201		{object}	contact_entity.Contact			"Created contact"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/contact [post]
func CreateContact(c *fiber.Ctx) error {
	var newContact contact_model.CreateContact
	if err := c.BodyParser(&newContact); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	contact, err := repository.Create(
		contact_entity.Contact{
			Name:      newContact.Name,
			Email:     newContact.Email,
			PhotoPath: newContact.PhotoPath,
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to create contact", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(contact)
}

// UpdateContact updates an existing contact.
//	@Summary		Edit a contact
//	@Description	Updates a contact using the provided data and returns the updated object.
//	@Tags			Contact
//	@Accept			json
//	@Produce		json
//	@Param			contact	body		contact_model.UpdateContact		true	"Contact data"
//	@Success		200		{object}	contact_entity.Contact			"Edited contact"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/contact [put]
func UpdateContact(c *fiber.Ctx) error {
	var editContact contact_model.UpdateContact
	if err := c.BodyParser(&editContact); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	contact, err := repository.Updates(
		contact_entity.Contact{
			Name:      editContact.Name,
			Email:     editContact.Email,
			PhotoPath: editContact.PhotoPath,
		},
		&contact_entity.Contact{
			Audit: common_model.Audit{
				Id: editContact.Id,
			},
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update contact", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(contact)
}
