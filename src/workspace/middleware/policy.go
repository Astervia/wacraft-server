package workspace_middleware

import (
	"slices"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	"github.com/gofiber/fiber/v2"
)

// RequirePolicy creates a middleware that checks if the user has any of the required policies.
// The user must have at least one of the specified policies to proceed.
// workspace.admin policy grants access to all operations.
func RequirePolicy(policies ...workspace_model.Policy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userPolicies := GetWorkspacePolicies(c)
		if userPolicies == nil {
			return c.Status(fiber.StatusForbidden).JSON(
				common_model.NewApiError("No policies found for user", nil, "middleware").Send(),
			)
		}

		// Check if user has workspace.admin (grants all access)
		if slices.Contains(userPolicies, workspace_model.PolicyWorkspaceAdmin) {
			return c.Next()
		}

		// Check if user has any of the required policies
		for _, required := range policies {
			if slices.Contains(userPolicies, required) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(
			common_model.NewApiError("Insufficient permissions", nil, "middleware").Send(),
		)
	}
}

// HasPolicy checks if the user has a specific policy
func HasPolicy(c *fiber.Ctx, policy workspace_model.Policy) bool {
	userPolicies := GetWorkspacePolicies(c)
	if userPolicies == nil {
		return false
	}

	// workspace.admin grants all access
	for _, p := range userPolicies {
		if p == workspace_model.PolicyWorkspaceAdmin {
			return true
		}
		if p == policy {
			return true
		}
	}

	return false
}
