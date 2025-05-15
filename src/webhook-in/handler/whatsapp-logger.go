package webhook_handler

import (
	"fmt"

	format_service "github.com/Astervia/wacraft-core/src/format/service"
	webhook_model "github.com/Rfluid/whatsapp-cloud-api/src/webhook/model"
	"github.com/gofiber/fiber/v2"
	"github.com/pterm/pterm"
)

func LoggerHandler(ctx *fiber.Ctx, body *webhook_model.WebhookBody) error {
	jsonAsString, err := format_service.Json(body)
	if err != nil {
		pterm.DefaultLogger.Error(
			fmt.Sprintf("Unable to format json: %v", err),
		)
		return err
	}

	pterm.DefaultLogger.Info(
		fmt.Sprintf("Body: %s", jsonAsString),
	)

	return nil
}
