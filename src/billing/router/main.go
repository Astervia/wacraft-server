package billing_router

import (
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_handler "github.com/Astervia/wacraft-server/src/billing/handler"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/billing")

	planRoutes(group)
	subscriptionRoutes(group)
	usageRoutes(group)
	endpointWeightRoutes(group)
	webhookRoutes(group)
}

func planRoutes(group fiber.Router) {
	plan := group.Group("/plan")

	// List plans - any authenticated user can see available plans
	plan.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.GetPlans)

	// Create plan - admin only (workspace-scoped with billing.admin policy)
	plan.Post("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyBillingAdmin),
		billing_middleware.ThroughputMiddleware,
		billing_handler.CreatePlan)

	// Update plan - admin only
	plan.Put("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyBillingAdmin),
		billing_middleware.ThroughputMiddleware,
		billing_handler.UpdatePlan)

	// Delete plan - admin only
	plan.Delete("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyBillingAdmin),
		billing_middleware.ThroughputMiddleware,
		billing_handler.DeletePlan)
}

func subscriptionRoutes(group fiber.Router) {
	sub := group.Group("/subscription")

	// List own subscriptions
	sub.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.GetSubscriptions)

	// Initiate checkout for a plan
	sub.Post("/checkout",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.Checkout)

	// Create manual subscription - admin only
	sub.Post("/manual",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyBillingAdmin),
		billing_middleware.ThroughputMiddleware,
		billing_handler.CreateManualSubscription)

	// Sync subscription state from payment provider
	sub.Post("/sync",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.SyncSubscription)

	// Reactivate subscription (undo pending cancellation)
	sub.Post("/reactivate",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.ReactivateSubscription)

	// Cancel subscription
	sub.Delete("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.CancelSubscription)
}

func usageRoutes(group fiber.Router) {
	// Get current usage - authenticated user, optional workspace context
	group.Get("/usage",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.OptionalWorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.GetUsage)
}

func endpointWeightRoutes(group fiber.Router) {
	ew := group.Group("/endpoint-weight")

	// List endpoint weights - admin only
	ew.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyBillingAdmin),
		billing_middleware.ThroughputMiddleware,
		billing_handler.GetEndpointWeights)

	// Create endpoint weight - admin only
	ew.Post("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyBillingAdmin),
		billing_middleware.ThroughputMiddleware,
		billing_handler.CreateEndpointWeight)

	// Delete endpoint weight - admin only
	ew.Delete("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.WorkspaceMiddleware,
		workspace_middleware.RequirePolicy(workspace_model.PolicyBillingAdmin),
		billing_middleware.ThroughputMiddleware,
		billing_handler.DeleteEndpointWeight)
}

func webhookRoutes(group fiber.Router) {
	// Stripe webhook - no auth (Stripe validates via signature)
	group.Post("/webhook/stripe", billing_handler.StripeWebhook)
}
