package whatsapp

import (
	"fmt"
	"os"

	"github.com/Astervia/wacraft-server/src/config/env"
	bootstrap_module "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap/model"
	bootstrap_service "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap/service"
	"github.com/pterm/pterm"
)

var WabaApi bootstrap_module.WhatsAppAPI

func Load() {
	pterm.DefaultLogger.Info("Loading WhatsApp integration...")
	var err error
	var wabaApi *bootstrap_module.WhatsAppAPI
	version := "v24.0"
	wabaApi, err = bootstrap_service.GenerateWhatsAppAPI(env.WabaAccessToken, &version, nil)
	if err != nil {
		pterm.DefaultLogger.Error(
			fmt.Sprintf("Unable to generate api: %v", err),
		)
		os.Exit(1)
	}
	WabaApi = *wabaApi
	_, err = WabaApi.SetWABAID(env.WabaID)
	if err != nil {
		pterm.DefaultLogger.Error(
			fmt.Sprintf("Unable to set WABA id: %v", err),
		)
		os.Exit(1)
	}
	_, err = WabaApi.SetWABAAccountID(env.WabaAccountID)
	if err != nil {
		pterm.DefaultLogger.Error(
			fmt.Sprintf("Unable to set WABA account id: %v", err),
		)
		os.Exit(1)
	}
	WabaApi.SetJSONHeaders().SetFormHeaders().SetWABAIDURL(nil)
	WabaApi.SetWABAAccountIDURL(nil)

	pterm.DefaultLogger.Info("WhatsApp integration loaded")
}
