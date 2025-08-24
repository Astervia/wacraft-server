package messaging_product_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// BlockContact blocks a messaging product contact by ID.
//
//	@Summary		Block messaging product contact
//	@Description	Blocks a messaging product contact by ID. Messages from this contact will be ignored.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			contact	body		common_model.RequiredID								true	"Contact ID to block"
//	@Success		201		{object}	messaging_product_entity.MessagingProductContact	"Blocked contact"
//	@Failure		400		{object}	common_model.DescriptiveError						"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError						"Failed to block contact"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact/block [patch]
func BlockContact(c *fiber.Ctx) error {
	var data common_model.RequiredID
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	updated, err := repository.Updates(
		messaging_product_entity.MessagingProductContact{
			Blocked: true,
		},
		&messaging_product_entity.MessagingProductContact{
			Audit: common_model.Audit{ID: data.ID},
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
//
//	@Summary		Unblock messaging product contact
//	@Description	Unblocks a messaging product contact by ID so it can send messages again.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			contact	body		common_model.RequiredID								true	"Contact ID to unblock"
//	@Success		201		{object}	messaging_product_entity.MessagingProductContact	"Unblocked contact"
//	@Failure		400		{object}	common_model.DescriptiveError						"Invalid request body"
//	@Failure		500		{object}	common_model.DescriptiveError						"Failed to unblock contact"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact/block [delete]
func UnblockContact(c *fiber.Ctx) error {
	var data common_model.RequiredID
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	updated, err := repository.Updates(
		map[string]any{
			"blocked": false,
		},
		&messaging_product_entity.MessagingProductContact{
			Audit: common_model.Audit{ID: data.ID},
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to unblock contact", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusCreated).JSON(updated)
}
