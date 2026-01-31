package phone_config_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	phone_config_handler "github.com/Astervia/wacraft-server/src/phone-config/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// Route registers phone config routes under /workspace/:workspace_id/phone-config
func Route(workspaceGroup fiber.Router) {
	group := workspaceGroup.Group("/phone-config")

	// List phone configs - requires phone_config.read policy
	group.Get("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyPhoneConfigRead),
		phone_config_handler.Get,
	)

	// Create phone config - requires phone_config.manage policy
	group.Post("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyPhoneConfigManage),
		phone_config_handler.Create,
	)

	// Get phone config by ID
	group.Get("/:id",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyPhoneConfigRead),
		phone_config_handler.GetByID,
	)

	// Update phone config
	group.Patch("/:id",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyPhoneConfigManage),
		phone_config_handler.Update,
	)

	// Delete phone config
	group.Delete("/:id",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyPhoneConfigManage),
		phone_config_handler.Delete,
	)
}
