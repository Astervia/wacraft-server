package env

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pterm/pterm"
)

var (
	BillingEnabled        bool   // Master toggle for throughput enforcement
	StripeSecretKey       string
	StripeWebhookSecret   string
	DefaultFreeThroughput int // Weighted requests per window for fallback free plan
	DefaultFreeWindow     int // Window in seconds for fallback free plan
)

func loadBillingEnv() {
	BillingEnabled = os.Getenv("BILLING_ENABLED") == "true"

	StripeSecretKey = os.Getenv("STRIPE_SECRET_KEY")
	StripeWebhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")

	DefaultFreeThroughput = 100
	if val := os.Getenv("DEFAULT_FREE_PLAN_THROUGHPUT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			DefaultFreeThroughput = parsed
		}
	}

	DefaultFreeWindow = 60
	if val := os.Getenv("DEFAULT_FREE_PLAN_WINDOW"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			DefaultFreeWindow = parsed
		}
	}

	if BillingEnabled {
		pterm.DefaultLogger.Info("Billing throughput enforcement is ENABLED")
	} else {
		pterm.DefaultLogger.Info("Billing throughput enforcement is DISABLED")
	}

	if StripeSecretKey != "" {
		pterm.DefaultLogger.Info("Stripe billing integration is CONFIGURED")
	} else if BillingEnabled {
		pterm.DefaultLogger.Warn("Stripe billing integration is NOT configured (STRIPE_SECRET_KEY not set)")
	}

	pterm.DefaultLogger.Info(
		fmt.Sprintf("Default free plan: %d weighted req/%ds", DefaultFreeThroughput, DefaultFreeWindow),
	)
}
