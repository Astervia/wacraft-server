package media_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	media_handler "github.com/Astervia/wacraft-server/src/media/handler"
	"github.com/gofiber/fiber/v2"
)

func whatsappRoutes(group fiber.Router) {
	waGroup := group.Group("/whatsapp")
	waGroup.Get("/:mediaID", auth_middleware.UserMiddleware, media_handler.GetWhatsAppMediaURL)
	waGroup.Get("/download/:mediaID", auth_middleware.UserMiddleware, media_handler.DownloadWhatsAppMedia)
	waGroup.Post("/media-info/download", auth_middleware.UserMiddleware, media_handler.DownloadFromMediaInfo)
	waGroup.Post("/upload", auth_middleware.UserMiddleware, media_handler.UploadWhatsAppMedia)
}
