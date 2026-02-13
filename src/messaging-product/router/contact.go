package messaging_product_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	messaging_product_handler "github.com/Astervia/wacraft-server/src/messaging-product/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func contactRoutes(group fiber.Router) {
	contactGroup := group.Group("/contact")

	mainContactRoutes(contactGroup)
	whatsAppContactRoutes(contactGroup)
}

func mainContactRoutes(contactGroup fiber.Router) {
	contactGroup.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactRead),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.GetContact)

	contactGroup.Get("/whatsapp",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactRead),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.GetWhatsAppContact)

	contactGroup.Post("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactManage),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.CreateContact)

	contactGroup.Delete("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactManage),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.DeleteContact)

	contactGroup.Patch("/block",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactManage),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.BlockContact)

	contactGroup.Delete("/block",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactManage),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.UnblockContact)

	contactGroup.Get("/content/like/:likeText",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactRead),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.ContactContentLike)

	contactGroup.Get("/count/content/like/:likeText",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactRead),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.ContactContentLikeCount)

	contactGroup.Put("/last-read-at/:messagingProductContactID",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactManage),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.UpdateContactLastReadAt)
}

func whatsAppContactRoutes(contactGroup fiber.Router) {
	wppGroup := contactGroup.Group("/whatsapp")
	wppGroup.Post("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactManage),
		billing_middleware.ThroughputMiddleware,
		messaging_product_handler.CreateWhatsAppContact)
}
