package billing_middleware

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	billing_service "github.com/Astervia/wacraft-server/src/billing/service"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// fallbackRoutes defines routes that fall back to a separate counter with the
// default free plan's throughput limits when the user's subscription limit is
// exceeded. This ensures users can always access routes needed to upgrade.
var fallbackRoutes = []struct {
	Method     string // Empty matches any method
	PathPrefix string
}{
	{"", "/billing/plan"},
	{"", "/billing/subscription"},
	{"GET", "/billing/usage"},
	{"GET", "/workspace"},
	{"GET", "/user/me"},
}

// isFallbackRoute checks if the request matches a route that should fall back
// to default free plan limits instead of returning 429.
func isFallbackRoute(method, path string) bool {
	for _, r := range fallbackRoutes {
		if (r.Method == "" || r.Method == method) && strings.HasPrefix(path, r.PathPrefix) {
			return true
		}
	}
	return false
}

// ThroughputMiddleware enforces throughput limits based on the user's active subscriptions.
// When BILLING_ENABLED=false (default), this middleware is a complete no-op.
// It must run after UserMiddleware.
//
// Scope priority per request:
//  1. Workspace — if a workspace is present in context and has available budget.
//  2. User — if no workspace or the workspace budget is exceeded.
//  3. Fallback — billing/upgrade routes get a separate budget when the active scope is exceeded.
func ThroughputMiddleware(c *fiber.Ctx) error {
	// Fast path: billing disabled — skip all logic
	if !env.BillingEnabled {
		return c.Next()
	}

	// Skip if user is not authenticated (auth routes, webhooks, etc.)
	user, ok := c.Locals("user").(*user_entity.User)
	if !ok || user == nil {
		return c.Next()
	}

	method := c.Method()
	path := c.Path()
	weight := billing_service.GetEndpointWeight(method, path)

	// Try workspace scope first if available.
	workspace, wsOk := c.Locals("workspace").(*workspace_entity.Workspace)
	if wsOk && workspace != nil {
		wsInfo := billing_service.ResolveThroughput(billing_model.ScopeWorkspace, nil, &workspace.ID)

		if wsInfo.Unlimited {
			scopeID := billing_service.ScopeKeyID(billing_model.ScopeWorkspace, nil, &workspace.ID)
			setUnlimitedRateLimitHeaders(c)
			setScopeHeaders(c, billing_model.ScopeWorkspace, scopeID, false)
			return c.Next()
		}

		scopeID := billing_service.ScopeKeyID(billing_model.ScopeWorkspace, nil, &workspace.ID)
		wsKey := billing_service.Key(string(billing_model.ScopeWorkspace), scopeID)
		wsCount := billing_service.GlobalCounter.Increment(wsKey, wsInfo.WindowSeconds, weight)

		if wsCount <= int64(wsInfo.Limit) {
			// Workspace has budget — charge it and return.
			c.Set("X-RateLimit-Limit", strconv.Itoa(wsInfo.Limit))
			c.Set("X-RateLimit-Remaining", strconv.FormatInt(max(0, int64(wsInfo.Limit)-wsCount), 10))
			resetTime := billing_service.GlobalCounter.WindowReset(wsKey)
			if !resetTime.IsZero() {
				c.Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
			}
			setScopeHeaders(c, billing_model.ScopeWorkspace, scopeID, false)
			return c.Next()
		}

		// Workspace exceeded — fall through to user scope.
	}

	return enforceScope(c, billing_model.ScopeUser, &user.ID, nil, method, path, weight)
}

// enforceScope resolves throughput for the given scope, increments the counter,
// and handles fallback routes. It sets rate limit headers on the response.
func enforceScope(c *fiber.Ctx, scope billing_model.Scope, userID *uuid.UUID, workspaceID *uuid.UUID, method, path string, weight int) error {
	info := billing_service.ResolveThroughput(scope, userID, workspaceID)

	scopeID := billing_service.ScopeKeyID(scope, userID, workspaceID)
	fallback := false

	if info.Unlimited {
		setUnlimitedRateLimitHeaders(c)
		setScopeHeaders(c, scope, scopeID, fallback)
		return c.Next()
	}

	key := billing_service.Key(string(scope), scopeID)
	count := billing_service.GlobalCounter.Increment(key, info.WindowSeconds, weight)

	if count > int64(info.Limit) {
		if !isFallbackRoute(method, path) {
			setScopeHeaders(c, scope, scopeID, fallback)
			return throughputExceeded(c, info, key)
		}
		// Fallback: use a separate counter with default free plan limits
		freeInfo := billing_service.DefaultFreeInfo()
		fallback = true
		if freeInfo.Unlimited {
			setUnlimitedRateLimitHeaders(c)
			setScopeHeaders(c, scope, scopeID, fallback)
			return c.Next()
		}
		fallbackKey := billing_service.Key(string(scope)+"-fallback", scopeID)
		fallbackCount := billing_service.GlobalCounter.Increment(fallbackKey, freeInfo.WindowSeconds, weight)
		if fallbackCount > int64(freeInfo.Limit) {
			setScopeHeaders(c, scope, scopeID, fallback)
			return throughputExceeded(c, freeInfo, fallbackKey)
		}
		info = freeInfo
		count = fallbackCount
		key = fallbackKey
	}

	// Set rate limit headers
	c.Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
	c.Set("X-RateLimit-Remaining", strconv.FormatInt(max(0, int64(info.Limit)-count), 10))
	resetTime := billing_service.GlobalCounter.WindowReset(key)
	if !resetTime.IsZero() {
		c.Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
	}
	setScopeHeaders(c, scope, scopeID, fallback)

	return c.Next()
}

// setScopeHeaders adds headers that describe which scope was charged.
func setScopeHeaders(c *fiber.Ctx, scope billing_model.Scope, scopeID string, fallback bool) {
	c.Set("X-RateLimit-Scope", string(scope))
	c.Set("X-RateLimit-Scope-ID", scopeID)
	if fallback {
		c.Set("X-RateLimit-Fallback", "true")
	}
}

func throughputExceeded(c *fiber.Ctx, info billing_service.ThroughputInfo, key string) error {
	resetTime := billing_service.GlobalCounter.WindowReset(key)
	retryAfter := time.Until(resetTime).Seconds()
	if retryAfter < 1 {
		retryAfter = 1
	}

	c.Set("Retry-After", strconv.Itoa(int(retryAfter)))
	c.Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
	c.Set("X-RateLimit-Remaining", "0")
	c.Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

	return c.Status(fiber.StatusTooManyRequests).JSON(
		common_model.NewApiError(
			fmt.Sprintf("Throughput limit exceeded: %d weighted requests per %ds", info.Limit, info.WindowSeconds),
			nil,
			"billing",
		).Send(),
	)
}

func setUnlimitedRateLimitHeaders(c *fiber.Ctx) {
	c.Set("X-RateLimit-Limit", "0")
	c.Set("X-RateLimit-Remaining", "-1")
	c.Set("X-RateLimit-Reset", "0")
}
