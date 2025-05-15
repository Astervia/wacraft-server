package messaging_product_handler

import (
	"time"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// UpdateContactLastReadAt sets the `last_read_at` field of a messaging product contact to the current timestamp.
//	@Summary		Sets last_read_at of the messaging_product_contact
//	@Description	Sets the `last_read_at` timestamp of the contact as the current date and time.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			messagingProductContactId	path		string												true	"Messaging product contact ID"
//	@Success		200							{object}	messaging_product_entity.MessagingProductContact	"Updated messaging product contact"
//	@Failure		400							{object}	common_model.DescriptiveError						"Invalid contact ID format"
//	@Failure		500							{object}	common_model.DescriptiveError						"Failed to update last_read_at"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact/last-read-at/{messagingProductContactId} [put]
func UpdateContactLastReadAt(c *fiber.Ctx) error {
	mpcId, err := uuid.Parse(c.Params("messagingProductContactId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to parse messaging product contact id string to UUID", err, "github.com/google/uuid"),
		)
	}

	mps, err := repository.Updates(
		messaging_product_entity.MessagingProductContact{
			Audit: common_model.Audit{
				Id: mpcId,
			},
			LastReadAt: time.Now(),
		},
		&messaging_product_entity.MessagingProductContact{
			Audit: common_model.Audit{
				Id: mpcId,
			},
		}, database.DB,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("unable to update messaging product contact last_read_at", err, "repository").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(mps)
}
