package main

import (
	_ "github.com/Astervia/wacraft-server/src/config"
	_ "github.com/Astervia/wacraft-server/src/database"
	_ "github.com/Astervia/wacraft-server/src/database/migrate"
	_ "github.com/Astervia/wacraft-server/src/integration"
	_ "github.com/Astervia/wacraft-server/src/server"
)

// @title						wacraft Server API
// @version					0.1.0
// @description				Backend server for the wacraft project. Handles WhatsApp Cloud API operations, including message sending, receiving, and webhook handling.
// @contact.name				Astervia Dev Team
// @contact.url				https://github.com/Astervia
// @contact.email				wacraft@astervia.tech
// @license.name				MIT
// @license.url				https://opensource.org/licenses/MIT
// @host						localhost:6900
// @BasePath					/
// @schemes					http https
// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						Authorization
func main() {
}
