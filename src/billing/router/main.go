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
	planPriceRoutes(group)
	subscriptionRoutes(group)
	paymentRoutes(group)
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

	// Create plan - superuser only (plans are platform-global, not workspace-scoped)
	plan.Post("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		auth_middleware.SuMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.CreatePlan)

	// Update plan - superuser only (plans are platform-global, not workspace-scoped)
	plan.Put("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		auth_middleware.SuMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.UpdatePlan)

	// Delete plan - superuser only (plans are platform-global, not workspace-scoped)
	plan.Delete("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		auth_middleware.SuMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.DeletePlan)
}

func planPriceRoutes(group fiber.Router) {
	price := group.Group("/plan/:plan_id/price")

	// List prices - any authenticated user can see a plan's prices
	price.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.GetPlanPrices)

	// Create price - superuser only (plan prices are platform-global, not workspace-scoped)
	price.Post("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		auth_middleware.SuMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.CreatePlanPrice)

	// Update price - superuser only (plan prices are platform-global, not workspace-scoped)
	price.Put("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		auth_middleware.SuMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.UpdatePlanPrice)

	// Delete price - superuser only (plan prices are platform-global, not workspace-scoped)
	price.Delete("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		auth_middleware.SuMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.DeletePlanPrice)
}

func subscriptionRoutes(group fiber.Router) {
	sub := group.Group("/subscription")

	// List subscriptions – user-scoped by default, workspace-scoped when X-Workspace-ID is provided
	sub.Get("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.OptionalWorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.GetSubscriptions)

	// Initiate checkout for a plan
	sub.Post("/checkout",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.OptionalWorkspaceMiddleware,
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
		workspace_middleware.OptionalWorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.SyncSubscription)

	// Reactivate subscription (undo pending cancellation)
	sub.Post("/reactivate",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.OptionalWorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.ReactivateSubscription)

	// Retry subscription payment
	sub.Post("/retry",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.OptionalWorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.RetrySubscription)

	// Resume checkout for a pending (unpaid) subscription
	sub.Post("/checkout-url",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.OptionalWorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.ResumeCheckout)

	// Cancel subscription
	sub.Delete("/",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.OptionalWorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.CancelSubscription)
}

func paymentRoutes(group fiber.Router) {
	// List payments – user-scoped by default, workspace-scoped when X-Workspace-ID is provided
	group.Get("/payment",
		auth_middleware.UserMiddleware,
		auth_middleware.EmailVerifiedMiddleware,
		workspace_middleware.OptionalWorkspaceMiddleware,
		billing_middleware.ThroughputMiddleware,
		billing_handler.GetPayments)
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
