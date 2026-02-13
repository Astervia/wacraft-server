package workspace_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	phone_config_router "github.com/Astervia/wacraft-server/src/phone-config/router"
	workspace_handler "github.com/Astervia/wacraft-server/src/workspace/handler"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/workspace")

	mainRoutes(group)
	workspaceRoutes(group)
	memberRoutes(group)
	invitationRoutes(group)

	// Phone config routes under /workspace/:workspace_id/phone-config
	phoneConfigGroup := group.Group("/:workspace_id")
	phone_config_router.Route(phoneConfigGroup)
}

// Routes that don't require workspace context
func mainRoutes(group fiber.Router) {
	// Create workspace - any authenticated user can create
	group.Post("", auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware, billing_middleware.ThroughputMiddleware, workspace_handler.Create)

	// List user's workspaces
	group.Get("", auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware, billing_middleware.ThroughputMiddleware, workspace_handler.Get)
}

// Routes that require workspace context
func workspaceRoutes(group fiber.Router) {
	// Get workspace by ID
	group.Get("/:workspace_id",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		workspace_handler.GetByID,
	)

	// Update workspace - requires workspace.settings policy
	group.Patch("/:workspace_id",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWorkspaceSettings),
		billing_middleware.ThroughputMiddleware,
		workspace_handler.Update,
	)

	// Delete workspace - requires workspace.admin policy
	group.Delete("/:workspace_id",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWorkspaceAdmin),
		billing_middleware.ThroughputMiddleware,
		workspace_handler.Delete,
	)
}

// Member management routes
func memberRoutes(group fiber.Router) {
	memberGroup := group.Group("/:workspace_id/member")

	// Add member - requires workspace.members policy
	memberGroup.Post("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWorkspaceMembers),
		billing_middleware.ThroughputMiddleware,
		workspace_handler.AddMember,
	)

	// List members - any workspace member can view
	memberGroup.Get("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		workspace_handler.GetMembers,
	)

	// Update member policies - requires workspace.members policy
	memberGroup.Patch("/:user_id",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWorkspaceMembers),
		billing_middleware.ThroughputMiddleware,
		workspace_handler.UpdateMemberPolicies,
	)

	// Remove member - requires workspace.members policy
	memberGroup.Delete("/:user_id",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWorkspaceMembers),
		billing_middleware.ThroughputMiddleware,
		workspace_handler.RemoveMember,
	)
}

// Invitation routes
func invitationRoutes(group fiber.Router) {
	invitationGroup := group.Group("/:workspace_id/invitation")

	// Create invitation - requires workspace.members policy
	invitationGroup.Post("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWorkspaceMembers),
		billing_middleware.ThroughputMiddleware,
		workspace_handler.CreateInvitation,
	)

	// List invitations - requires workspace.members policy
	invitationGroup.Get("",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWorkspaceMembers),
		billing_middleware.ThroughputMiddleware,
		workspace_handler.GetInvitations,
	)

	// Revoke invitation - requires workspace.members policy
	invitationGroup.Delete("/:invitation_id",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyWorkspaceMembers),
		billing_middleware.ThroughputMiddleware,
		workspace_handler.RevokeInvitation,
	)
}
