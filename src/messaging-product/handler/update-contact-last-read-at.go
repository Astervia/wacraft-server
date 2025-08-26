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
//
//	@Summary		Update last_read_at timestamp
//	@Description	Sets the `last_read_at` field of the specified messaging product contact to the current timestamp.
//	@Tags			Messaging product contact
//	@Accept			json
//	@Produce		json
//	@Param			messagingProductContactID	path		string												true	"Messaging product contact ID"
//	@Success		200							{object}	messaging_product_entity.MessagingProductContact	"Updated messaging product contact"
//	@Failure		400							{object}	common_model.DescriptiveError						"Invalid contact ID format"
//	@Failure		500							{object}	common_model.DescriptiveError						"Failed to update last_read_at"
//	@Security		ApiKeyAuth
//	@Router			/messaging-product/contact/last-read-at/{messagingProductContactID} [put]
func UpdateContactLastReadAt(c *fiber.Ctx) error {
	mpcID, err := uuid.Parse(c.Params("messagingProductContactID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("unable to parse messaging product contact id string to UUID", err, "github.com/google/uuid"),
		)
	}

	now := time.Now()
	mps, err := repository.Updates(
		messaging_product_entity.MessagingProductContact{
			Audit: common_model.Audit{
				ID: mpcID,
			},
			LastReadAt: &now,
		},
		&messaging_product_entity.MessagingProductContact{
			Audit: common_model.Audit{
				ID: mpcID,
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
