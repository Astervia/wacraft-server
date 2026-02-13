package billing_handler

import (
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	billing_service "github.com/Astervia/wacraft-server/src/billing/service"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetUsage returns current throughput usage for all active scopes.
// User-scoped usage is always included. If X-Workspace-ID is provided,
// workspace-scoped usage is appended.
//
//	@Summary		Get throughput usage
//	@Description	Returns current throughput usage. Always includes user-scoped usage. If X-Workspace-ID is provided, also includes workspace-scoped usage.
//	@Tags			Billing Usage
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		billing_model.UsageSummary		"Usage summaries"
//	@Security		ApiKeyAuth
//	@Router			/billing/usage [get]
func GetUsage(c *fiber.Ctx) error {
	user := workspace_middleware.GetUser(c)
	workspace := workspace_middleware.GetWorkspace(c)

	// Always include user-scoped usage
	summaries := buildScopeUsage(billing_model.ScopeUser, &user.ID, nil)

	// Append workspace-scoped usage when workspace is available
	if workspace != nil {
		summaries = append(summaries, buildScopeUsage(billing_model.ScopeWorkspace, nil, &workspace.ID)...)
	}

	return c.Status(fiber.StatusOK).JSON(summaries)
}

// buildScopeUsage builds usage summaries for a single scope, including fallback
// budget when the main limit is exceeded.
func buildScopeUsage(scope billing_model.Scope, userID *uuid.UUID, workspaceID *uuid.UUID) []billing_model.UsageSummary {
	info := billing_service.ResolveThroughput(scope, userID, workspaceID)

	if info.Unlimited {
		return []billing_model.UsageSummary{{
			Scope:       scope,
			UserID:      userID,
			WorkspaceID: workspaceID,
			Unlimited:   true,
			Remaining:   -1,
		}}
	}

	scopeID := billing_service.ScopeKeyID(scope, userID, workspaceID)
	key := billing_service.Key(string(scope), scopeID)
	current := billing_service.GlobalCounter.Current(key)
	remaining := max(int64(info.Limit)-current, 0)

	summaries := []billing_model.UsageSummary{{
		Scope:           scope,
		UserID:          userID,
		WorkspaceID:     workspaceID,
		ThroughputLimit: info.Limit,
		WindowSeconds:   info.WindowSeconds,
		CurrentUsage:    current,
		Remaining:       remaining,
	}}

	// Include fallback budget when the user scope's main limit is exceeded.
	// Only user scope has fallback — workspace exceeded cascades to user instead.
	if scope == billing_model.ScopeUser && current > int64(info.Limit) {
		freeInfo := billing_service.DefaultFreeInfo()
		if freeInfo.Unlimited {
			summaries = append(summaries, billing_model.UsageSummary{
				Scope:       scope,
				UserID:      userID,
				WorkspaceID: workspaceID,
				Unlimited:   true,
				Remaining:   -1,
				Fallback:    true,
			})
		} else {
			fallbackKey := billing_service.Key(string(scope)+"-fallback", scopeID)
			fallbackCurrent := billing_service.GlobalCounter.Current(fallbackKey)
			fallbackRemaining := max(int64(freeInfo.Limit)-fallbackCurrent, 0)

			summaries = append(summaries, billing_model.UsageSummary{
				Scope:           scope,
				UserID:          userID,
				WorkspaceID:     workspaceID,
				ThroughputLimit: freeInfo.Limit,
				WindowSeconds:   freeInfo.WindowSeconds,
				CurrentUsage:    fallbackCurrent,
				Remaining:       fallbackRemaining,
				Fallback:        true,
			})
		}
	}

	return summaries
}
