package env

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pterm/pterm"
)

var (
	WabaID                   string
	WabaAccessToken          string
	WabaAccountID            string
	DisplayPhone             string
	MetaAppSecret            string
	MetaVerifyToken          string
	MessageStatusSyncTimeout = 20 * time.Second
)

func loadWhatsAppEnv() {
	WabaID = os.Getenv("WABA_ID")
	WabaAccessToken = os.Getenv("WABA_ACCESS_TOKEN")
	WabaAccountID = os.Getenv("WABA_ACCOUNT_ID")
	DisplayPhone = os.Getenv("DISPLAY_PHONE")
	MetaAppSecret = os.Getenv("META_APP_SECRET")
	MetaVerifyToken = os.Getenv("META_VERIFY_TOKEN")

	messageStatusSyncTimeoutSeconds := os.Getenv("MESSAGE_STATUS_SYNC_TIMEOUT_SECONDS")
	timeoutSecToInt, err := strconv.Atoi(messageStatusSyncTimeoutSeconds)
	if err == nil && timeoutSecToInt > 0 {
		MessageStatusSyncTimeout = time.Duration(timeoutSecToInt) * time.Second
	}

	pterm.DefaultLogger.Info(
		fmt.Sprintf(
			"WhatsApp environment done with waba id %s and message<=>status timeout %s seconds",
			WabaID,
			MessageStatusSyncTimeout,
		),
	)
}
