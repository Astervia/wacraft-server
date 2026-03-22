package billing_middleware

import (
	"net/http/httptest"
	"testing"

	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	billing_service "github.com/Astervia/wacraft-server/src/billing/service"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const testPhoneConfigKey = "phone_config"

// newWebhookInApp builds a minimal Fiber app that sets a phone config in locals
// then runs WebhookInThroughputMiddleware, responding 200 on success.
func newWebhookInApp(phoneConfig *phone_config_entity.PhoneConfig) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if phoneConfig != nil {
			c.Locals(testPhoneConfigKey, phoneConfig)
		}
		return c.Next()
	})
	app.Use(WebhookInThroughputMiddleware(testPhoneConfigKey))
	app.Post("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

// resetCounter replaces GlobalCounter with a fresh in-memory counter.
func resetCounter() {
	billing_service.SetThroughputCounter(
		billing_service.NewThroughputCounter(synch_service.NewMemoryCounter()),
	)
}

func TestWebhookInThroughput_BillingDisabled(t *testing.T) {
	env.BillingEnabled = false
	t.Cleanup(func() { env.BillingEnabled = false })

	wsID := uuid.New()
	pc := &phone_config_entity.PhoneConfig{WorkspaceID: &wsID}
	app := newWebhookInApp(pc)

	resp, err := app.Test(httptest.NewRequest("POST", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-RateLimit-Limit") != "" {
		t.Fatal("expected no rate limit headers when billing is disabled")
	}
}

func TestWebhookInThroughput_NoPhoneConfig(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })

	app := newWebhookInApp(nil) // no phone config set in locals

	resp, err := app.Test(httptest.NewRequest("POST", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-RateLimit-Limit") != "" {
		t.Fatal("expected no rate limit headers without a phone config")
	}
}

func TestWebhookInThroughput_NoWorkspaceID(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })

	pc := &phone_config_entity.PhoneConfig{WorkspaceID: nil}
	app := newWebhookInApp(pc)

	resp, err := app.Test(httptest.NewRequest("POST", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-RateLimit-Limit") != "" {
		t.Fatal("expected no rate limit headers when phone config has no workspace")
	}
}

func TestWebhookInThroughput_WithinLimit(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })
	resetCounter()
	t.Cleanup(resetCounter)

	const limit = 10
	wsID := uuid.New()
	billing_service.InvalidateCache(billing_model.ScopeWorkspace, wsID)
	billing_service.SetQueryThroughputFn(func(_ billing_model.Scope, _ *uuid.UUID, _ *uuid.UUID) billing_service.ThroughputInfo {
		return billing_service.ThroughputInfo{Limit: limit, WindowSeconds: 60}
	})
	t.Cleanup(func() { billing_service.SetQueryThroughputFn(nil) })

	pc := &phone_config_entity.PhoneConfig{WorkspaceID: &wsID}
	app := newWebhookInApp(pc)

	resp, err := app.Test(httptest.NewRequest("POST", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-RateLimit-Limit") == "" {
		t.Fatal("expected X-RateLimit-Limit header")
	}
	if resp.Header.Get("X-RateLimit-Remaining") == "" {
		t.Fatal("expected X-RateLimit-Remaining header")
	}
	if resp.Header.Get("X-RateLimit-Scope") != "workspace" {
		t.Fatalf("expected scope=workspace, got %q", resp.Header.Get("X-RateLimit-Scope"))
	}
	if resp.Header.Get("X-RateLimit-Scope-ID") != wsID.String() {
		t.Fatalf("expected scope ID %s, got %s", wsID, resp.Header.Get("X-RateLimit-Scope-ID"))
	}
}

func TestWebhookInThroughput_Exceeded(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })
	resetCounter()
	t.Cleanup(resetCounter)

	const limit = 2
	wsID := uuid.New()
	billing_service.InvalidateCache(billing_model.ScopeWorkspace, wsID)
	billing_service.SetQueryThroughputFn(func(_ billing_model.Scope, _ *uuid.UUID, _ *uuid.UUID) billing_service.ThroughputInfo {
		return billing_service.ThroughputInfo{Limit: limit, WindowSeconds: 60}
	})
	t.Cleanup(func() { billing_service.SetQueryThroughputFn(nil) })

	// Pre-charge the counter up to the limit
	scopeID := billing_service.ScopeKeyID(billing_model.ScopeWorkspace, nil, &wsID)
	key := billing_service.Key(string(billing_model.ScopeWorkspace), scopeID)
	for i := 0; i < limit; i++ {
		billing_service.GlobalCounter.Increment(key, 60, 1)
	}

	pc := &phone_config_entity.PhoneConfig{WorkspaceID: &wsID}
	app := newWebhookInApp(pc)

	resp, err := app.Test(httptest.NewRequest("POST", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429")
	}
}

func TestWebhookInThroughput_Unlimited(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })
	resetCounter()
	t.Cleanup(resetCounter)

	wsID := uuid.New()
	billing_service.InvalidateCache(billing_model.ScopeWorkspace, wsID)
	billing_service.SetQueryThroughputFn(func(_ billing_model.Scope, _ *uuid.UUID, _ *uuid.UUID) billing_service.ThroughputInfo {
		return billing_service.ThroughputInfo{Unlimited: true}
	})
	t.Cleanup(func() { billing_service.SetQueryThroughputFn(nil) })

	pc := &phone_config_entity.PhoneConfig{WorkspaceID: &wsID}
	app := newWebhookInApp(pc)

	resp, err := app.Test(httptest.NewRequest("POST", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-RateLimit-Limit") != "0" {
		t.Fatalf("expected X-RateLimit-Limit: 0 for unlimited, got %q", resp.Header.Get("X-RateLimit-Limit"))
	}
	if resp.Header.Get("X-RateLimit-Remaining") != "-1" {
		t.Fatalf("expected X-RateLimit-Remaining: -1 for unlimited, got %q", resp.Header.Get("X-RateLimit-Remaining"))
	}
}
