package workspace_middleware

import (
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
)

// WebSocketWorkspaceMiddleware extracts workspace ID from header or query param and validates user membership.
// This middleware should be used for WebSocket routes that require workspace context.
// It supports both X-Workspace-ID header and workspace_id query param since some WebSocket clients don't support custom headers.
//
// @Summary		Workspace WebSocket middleware
// @Description	Validates workspace membership for WebSocket connections using header or query param
// @Tags			Websocket
// @Accept			json
// @Produce		json
// @Success		200 "Workspace validated"
// @Router			/websocket/{channel} [get]
// @Security		ApiKeyAuth
func WebSocketWorkspaceMiddleware(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	// Get workspace ID from header or query param
	workspaceIDStr := c.Get("X-Workspace-ID")
	if workspaceIDStr == "" {
		workspaceIDStr = c.Query("workspace_id")
	}

	if workspaceIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "X-Workspace-ID header or workspace_id query param is required",
		})
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid workspace ID format",
		})
	}

	// Fetch workspace from database
	var workspace workspace_entity.Workspace
	if err := database.DB.First(&workspace, workspaceID).Error; err != nil {
		pterm.DefaultLogger.Warn("Workspace not found: " + err.Error())
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Workspace not found",
		})
	}

	// Validate user is a member of the workspace
	user, ok := c.Locals("user").(*user_entity.User)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "User not authenticated. WebSocketWorkspaceMiddleware requires UserMiddleware to run first.",
		})
	}

	var memberCount int64
	err = database.DB.Model(&workspace_entity.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, user.ID).
		Count(&memberCount).Error

	if err != nil {
		pterm.DefaultLogger.Error("Error checking workspace membership: " + err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error validating workspace membership",
		})
	}

	if memberCount == 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "You are not a member of this workspace",
		})
	}

	// Store workspace in context
	c.Locals("workspace", &workspace)

	return c.Next()
}
