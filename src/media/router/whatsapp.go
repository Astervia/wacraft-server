package media_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	media_handler "github.com/Astervia/wacraft-server/src/media/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func whatsappRoutes(group fiber.Router) {
	waGroup := group.Group("/whatsapp")
	waGroup.Get("/:mediaID", auth_middleware.UserMiddleware, auth_middleware.EmailVerifiedMiddleware, workspace_middleware.WorkspaceMiddleware, billing_middleware.ThroughputMiddleware, media_handler.GetWhatsAppMediaURL)
	waGroup.Get("/download/:mediaID", auth_middleware.UserMiddleware, auth_middleware.EmailVerifiedMiddleware, workspace_middleware.WorkspaceMiddleware, billing_middleware.ThroughputMiddleware, media_handler.DownloadWhatsAppMedia)
	waGroup.Post("/media-info/download", auth_middleware.UserMiddleware, auth_middleware.EmailVerifiedMiddleware, workspace_middleware.WorkspaceMiddleware, billing_middleware.ThroughputMiddleware, media_handler.DownloadFromMediaInfo)
	waGroup.Post("/upload", auth_middleware.UserMiddleware, auth_middleware.EmailVerifiedMiddleware, workspace_middleware.WorkspaceMiddleware, billing_middleware.ThroughputMiddleware, media_handler.UploadWhatsAppMedia)
}
