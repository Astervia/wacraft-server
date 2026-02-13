package contact_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	contact_handler "github.com/Astervia/wacraft-server/src/contact/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/contact")

	mainRoutes(group)
}

func mainRoutes(group fiber.Router) {
	group.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactRead),
		billing_middleware.ThroughputMiddleware,
		contact_handler.Get)
	group.Post("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactManage),
		billing_middleware.ThroughputMiddleware,
		contact_handler.CreateContact)
	group.Put("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactManage),
		billing_middleware.ThroughputMiddleware,
		contact_handler.UpdateContact)
	group.Delete("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyContactManage),
		billing_middleware.ThroughputMiddleware,
		contact_handler.DeleteContactByID)
}
