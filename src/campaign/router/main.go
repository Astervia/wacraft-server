package campaign_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	campaign_handler "github.com/Astervia/wacraft-server/src/campaign/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/campaign")

	mainRoutes(group)
	messageRoutes(group)
	errorRoutes(group)
}

func mainRoutes(group fiber.Router) {
	group.Get("",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRead),
		campaign_handler.Get)
	group.Post("",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignManage),
		campaign_handler.Create)
	group.Patch("",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignManage),
		campaign_handler.Update)
	group.Delete("",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignManage),
		campaign_handler.Delete)
	group.Get("/content/:keyName/like/:likeText",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRead),
		campaign_handler.ContentKeyLike)
}

func messageRoutes(group fiber.Router) {
	messageGroup := group.Group("/message")

	messageGroup.Get("",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRead),
		campaign_handler.GetMessages)
	messageGroup.Get("/sent",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRead),
		campaign_handler.GetSentMessages)
	messageGroup.Get("/unsent",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRead),
		campaign_handler.GetUnsentMessages)
	messageGroup.Post("",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRun),
		campaign_handler.CreateMessage)
	messageGroup.Delete("",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignManage),
		campaign_handler.DeleteMessage)
	messageGroup.Get("/count",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRead),
		campaign_handler.CountMessages)
	messageGroup.Get("/count/sent",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRead),
		campaign_handler.CountSentMessages)
	messageGroup.Get("/count/unsent",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRead),
		campaign_handler.CountUnsentMessages)
}

func errorRoutes(group fiber.Router) {
	messageGroup := group.Group("/error")

	messageGroup.Get("",
		auth_middleware.UserMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyCampaignRead),
		campaign_handler.GetErrors)
}
