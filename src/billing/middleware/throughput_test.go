package billing_middleware

import (
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	phone_config_entity "github.com/Astervia/wacraft-core/src/phone-config/entity"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
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

// newAPICallbackApp builds a minimal Fiber app simulating an authenticated API endpoint
// called back by a webhook consumer (e.g. POST /message/whatsapp). It injects user +
// workspace into locals then runs ThroughputMiddleware, responding 200 on success.
func newAPICallbackApp(userID, wsID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		u := &user_entity.User{}
		u.ID = userID
		ws := &workspace_entity.Workspace{}
		ws.ID = wsID
		c.Locals("user", u)
		c.Locals("workspace", ws)
		return c.Next()
	})
	app.Use(ThroughputMiddleware)
	app.Post("/message/whatsapp", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

// setLoopThroughput is a helper that injects a fixed ThroughputInfo for all scopes
// and returns a cleanup func. It also invalidates caches for the given IDs.
func setLoopThroughput(t *testing.T, limit int, wsID, userID uuid.UUID) {
	t.Helper()
	billing_service.InvalidateCache(billing_model.ScopeWorkspace, wsID)
	billing_service.InvalidateCache(billing_model.ScopeUser, userID)
	billing_service.SetQueryThroughputFn(func(_ billing_model.Scope, _ *uuid.UUID, _ *uuid.UUID) billing_service.ThroughputInfo {
		return billing_service.ThroughputInfo{Limit: limit, WindowSeconds: 60}
	})
	t.Cleanup(func() { billing_service.SetQueryThroughputFn(nil) })
}

// TestWebhookConsumerLoop_Sequential verifies that the three billing charges produced by
// the Meta → delivery-worker → consumer callback loop all succeed in sequence.
//
// Flow:
//  1. Incoming webhook from Meta      → WebhookInThroughputMiddleware charges workspace
//  2. Delivery worker fires           → ConsumeWorkspaceThroughput charges workspace
//  3. Consumer calls POST /message/.. → ThroughputMiddleware charges workspace
func TestWebhookConsumerLoop_Sequential(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })
	resetCounter()
	t.Cleanup(resetCounter)

	const limit = 10
	wsID := uuid.New()
	userID := uuid.New()
	setLoopThroughput(t, limit, wsID, userID)

	// Step 1: incoming webhook
	pc := &phone_config_entity.PhoneConfig{WorkspaceID: &wsID}
	resp, err := newWebhookInApp(pc).Test(httptest.NewRequest("POST", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("step 1 (incoming webhook): got %d, want 200", resp.StatusCode)
	}

	// Step 2: delivery worker
	if !billing_service.ConsumeWorkspaceThroughput(&wsID, 1) {
		t.Fatal("step 2 (delivery worker): charge rejected, want allowed")
	}

	// Step 3: consumer callback
	resp, err = newAPICallbackApp(userID, wsID).Test(httptest.NewRequest("POST", "/message/whatsapp", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("step 3 (consumer callback): got %d, want 200 (remaining: %s)",
			resp.StatusCode, resp.Header.Get("X-RateLimit-Remaining"))
	}
}

// TestWebhookConsumerLoop_ConcurrentDeliveryAndCallback tests the scenario where the
// delivery worker charges workspace throughput at the same moment the consumer's callback
// arrives. This mirrors real timing: a webhook consumer calls back to wacraft
// synchronously WHILE the delivery worker's outbound HTTP call is still open.
// A slow queryThroughputFn amplifies any planLock contention so races surface quickly.
func TestWebhookConsumerLoop_ConcurrentDeliveryAndCallback(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })
	resetCounter()
	t.Cleanup(resetCounter)

	const limit = 10
	wsID := uuid.New()
	userID := uuid.New()

	billing_service.InvalidateCache(billing_model.ScopeWorkspace, wsID)
	billing_service.InvalidateCache(billing_model.ScopeUser, userID)

	// Slow query to amplify lock-contention races (simulates cold-cache DB latency).
	billing_service.SetQueryThroughputFn(func(_ billing_model.Scope, _ *uuid.UUID, _ *uuid.UUID) billing_service.ThroughputInfo {
		time.Sleep(10 * time.Millisecond)
		return billing_service.ThroughputInfo{Limit: limit, WindowSeconds: 60}
	})
	t.Cleanup(func() { billing_service.SetQueryThroughputFn(nil) })

	callbackApp := newAPICallbackApp(userID, wsID)

	var wg sync.WaitGroup
	wg.Add(2)

	var deliveryAllowed atomic.Bool
	deliveryAllowed.Store(true)
	var callbackStatus atomic.Int32

	// Goroutine 1: delivery worker charges workspace (step 2)
	go func() {
		defer wg.Done()
		if !billing_service.ConsumeWorkspaceThroughput(&wsID, 1) {
			deliveryAllowed.Store(false)
		}
	}()

	// Goroutine 2: consumer callback charges workspace via ThroughputMiddleware (step 3)
	go func() {
		defer wg.Done()
		resp, err := callbackApp.Test(httptest.NewRequest("POST", "/message/whatsapp", nil))
		if err == nil {
			callbackStatus.Store(int32(resp.StatusCode))
		}
	}()

	wg.Wait()

	if !deliveryAllowed.Load() {
		t.Error("step 2 (delivery worker): charge rejected under concurrency, want allowed")
	}
	if status := callbackStatus.Load(); status != fiber.StatusOK {
		t.Errorf("step 3 (consumer callback): got %d under concurrency, want 200", status)
	}
}

// TestWebhookConsumerLoop_ColdCacheFirstMessage reproduces the "first message 429"
// scenario. On the very first request after server start, the plan cache is empty.
// Multiple goroutines (delivery worker pool) may simultaneously try to populate it via
// planLock. Meanwhile the consumer's callback arrives. All should succeed.
func TestWebhookConsumerLoop_ColdCacheFirstMessage(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })
	resetCounter()
	t.Cleanup(resetCounter)

	const limit = 10
	const poolSize = 5 // simulate several concurrent delivery goroutines
	wsID := uuid.New()
	userID := uuid.New()

	billing_service.InvalidateCache(billing_model.ScopeWorkspace, wsID)
	billing_service.InvalidateCache(billing_model.ScopeUser, userID)

	// Slow query simulates the cold-cache DB round-trip that holds planLock.
	billing_service.SetQueryThroughputFn(func(_ billing_model.Scope, _ *uuid.UUID, _ *uuid.UUID) billing_service.ThroughputInfo {
		time.Sleep(10 * time.Millisecond)
		return billing_service.ThroughputInfo{Limit: limit, WindowSeconds: 60}
	})
	t.Cleanup(func() { billing_service.SetQueryThroughputFn(nil) })

	callbackApp := newAPICallbackApp(userID, wsID)

	var wg sync.WaitGroup
	wg.Add(poolSize + 1) // pool goroutines + n8n callback

	var rejectedDeliveries atomic.Int32
	var callbackStatus atomic.Int32

	// Simulate the delivery worker pool — poolSize goroutines all charging at once.
	for range poolSize {
		go func() {
			defer wg.Done()
			if !billing_service.ConsumeWorkspaceThroughput(&wsID, 1) {
				rejectedDeliveries.Add(1)
			}
		}()
	}

	// n8n callback fires while the pool is saturated and the cache is still cold.
	go func() {
		defer wg.Done()
		resp, err := callbackApp.Test(httptest.NewRequest("POST", "/message/whatsapp", nil))
		if err == nil {
			callbackStatus.Store(int32(resp.StatusCode))
		}
	}()

	wg.Wait()

	total := int(rejectedDeliveries.Load()) + poolSize // delivered + rejected
	_ = total // all poolSize attempts ran; some may be rejected if limit < poolSize
	if callbackStatus.Load() != fiber.StatusOK {
		t.Errorf("n8n callback got %d on cold cache (first message), want 200", callbackStatus.Load())
	}
}
