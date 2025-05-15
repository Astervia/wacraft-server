package messaging_product_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// BlockContact blocks a messaging product contact by ID.
//	@Summary		Blocks a messaging product contact
//	@Description	Blocks a messaging product contact by ID. Messages from this contact will be ignored.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			contact	body		common_model.RequiredId								true	"Contact ID to block"
//	@Success		201		{object}	messaging_product_entity.MessagingProductContact	"Blocked contact"
//	@Failure		400		{object}	common_model.DescriptiveError						"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError						"Failed to block contact"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact/block [patch]
func BlockContact(c *fiber.Ctx) error {
	// Parse the request body
	var data common_model.RequiredId
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	updated, err := repository.Updates(
		messaging_product_entity.MessagingProductContact{
			Blocked: true,
		},
		&messaging_product_entity.MessagingProductContact{
			Audit: common_model.Audit{Id: data.Id},
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update contact", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(updated)
}

// UnblockContact unblocks a messaging product contact by ID.
//	@Summary		Unblocks a messaging product contact
//	@Description	Unblocks a messaging product contact by ID so it can send messages again.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			contact	body		common_model.RequiredId								true	"Contact ID to unblock"
//	@Success		201		{object}	messaging_product_entity.MessagingProductContact	"Unblocked contact"
//	@Failure		400		{object}	common_model.DescriptiveError						"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError						"Failed to unblock contact"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact/block [delete]
func UnblockContact(c *fiber.Ctx) error {
	// Parse the request body
	var data common_model.RequiredId
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	updated, err := repository.Updates(
		map[string]interface{}{
			"blocked": false,
		},
		&messaging_product_entity.MessagingProductContact{
			Audit: common_model.Audit{Id: data.Id},
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to unblock contact", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(updated)
}
