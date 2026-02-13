package workspace_middleware

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// WorkspaceMiddleware extracts X-Workspace-ID header, validates membership,
// and stores workspace + policies in context.
func WorkspaceMiddleware(c *fiber.Ctx) error {
	// Get workspace ID from header
	workspaceIDStr := c.Get("X-Workspace-ID")
	if workspaceIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("X-Workspace-ID header is required", nil, "middleware").Send(),
		)
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid workspace ID format", err, "middleware").Send(),
		)
	}

	// Get user from context (set by UserMiddleware)
	user, ok := c.Locals("user").(*user_entity.User)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("User not authenticated", nil, "middleware").Send(),
		)
	}

	// Fetch workspace
	var workspace workspace_entity.Workspace
	if err := database.DB.First(&workspace, workspaceID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Workspace not found", err, "middleware").Send(),
		)
	}

	// Fetch workspace membership
	var member workspace_entity.WorkspaceMember
	if err := database.DB.Where("workspace_id = ? AND user_id = ?", workspaceID, user.ID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(
			common_model.NewApiError("You are not a member of this workspace", err, "middleware").Send(),
		)
	}

	// Fetch member policies
	var policies []workspace_entity.WorkspaceMemberPolicy
	if err := database.DB.Where("workspace_member_id = ?", member.ID).Find(&policies).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to fetch workspace policies", err, "middleware").Send(),
		)
	}

	// Convert to policy strings for easier checking
	policyStrings := make([]workspace_model.Policy, len(policies))
	for i, p := range policies {
		policyStrings[i] = p.Policy
	}

	// Store in context
	c.Locals("workspace", &workspace)
	c.Locals("workspace_member", &member)
	c.Locals("workspace_policies", policyStrings)

	return c.Next()
}

// OptionalWorkspaceMiddleware behaves like WorkspaceMiddleware but skips
// silently when X-Workspace-ID is not provided. Useful for endpoints that
// optionally accept a workspace context (e.g. billing usage).
func OptionalWorkspaceMiddleware(c *fiber.Ctx) error {
	if c.Get("X-Workspace-ID") == "" {
		return c.Next()
	}
	return WorkspaceMiddleware(c)
}

// GetWorkspace retrieves the workspace from context
func GetWorkspace(c *fiber.Ctx) *workspace_entity.Workspace {
	workspace, ok := c.Locals("workspace").(*workspace_entity.Workspace)
	if !ok {
		return nil
	}
	return workspace
}

// GetWorkspaceMember retrieves the workspace member from context
func GetWorkspaceMember(c *fiber.Ctx) *workspace_entity.WorkspaceMember {
	member, ok := c.Locals("workspace_member").(*workspace_entity.WorkspaceMember)
	if !ok {
		return nil
	}
	return member
}

// GetWorkspacePolicies retrieves the workspace policies from context
func GetWorkspacePolicies(c *fiber.Ctx) []workspace_model.Policy {
	policies, ok := c.Locals("workspace_policies").([]workspace_model.Policy)
	if !ok {
		return nil
	}
	return policies
}

// GetUser retrieves the user from context (set by UserMiddleware)
func GetUser(c *fiber.Ctx) *user_entity.User {
	user, ok := c.Locals("user").(*user_entity.User)
	if !ok {
		return nil
	}
	return user
}
