package billing_service

import (
	"os"
	"sync"
	"testing"
	"time"

	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/google/uuid"
)

// resetGlobalCounter replaces GlobalCounter with a fresh in-memory counter.
func resetGlobalCounter() {
	SetThroughputCounter(NewThroughputCounter(synch_service.NewMemoryCounter()))
}

func TestThroughput_MemoryIncrement(t *testing.T) {
	c := NewThroughputCounter(synch_service.NewMemoryCounter())
	const key = "test:memory"
	const window = 60

	for i := 0; i < 10; i++ {
		c.Increment(key, window, 1)
	}

	got := c.Current(key, window)
	if got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
}

func TestThroughput_TTLExpiry(t *testing.T) {
	c := NewThroughputCounter(synch_service.NewMemoryCounter())
	const window = 1 // 1-second window

	key := "test:ttl"
	c.Increment(key, window, 5)

	if got := c.Current(key, window); got != 5 {
		t.Fatalf("before expiry: expected 5, got %d", got)
	}

	time.Sleep(1100 * time.Millisecond)

	if got := c.Current(key, window); got != 0 {
		t.Fatalf("after expiry: expected 0, got %d", got)
	}
}

func TestThroughput_RateLimitEnforced(t *testing.T) {
	c := NewThroughputCounter(synch_service.NewMemoryCounter())
	const limit = 10
	const window = 60
	key := "test:ratelimit"

	var rejected int
	for i := 0; i < 15; i++ {
		count := c.Increment(key, window, 1)
		if count > limit {
			rejected++
		}
	}

	if rejected != 5 {
		t.Fatalf("expected 5 rejected, got %d", rejected)
	}
}

func TestThroughput_RedisCrossInstance(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set — skipping Redis integration test")
	}

	clientA, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 15})
	if err != nil {
		t.Fatalf("clientA: %v", err)
	}
	clientB, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 15})
	if err != nil {
		t.Fatalf("clientB: %v", err)
	}

	// Flush DB 15 before the test.
	clientA.Redis().FlushDB(t.Context())
	t.Cleanup(func() { clientA.Redis().FlushDB(t.Context()) })

	cA := NewThroughputCounter(synch_redis.NewRedisCounter(clientA))
	cB := NewThroughputCounter(synch_redis.NewRedisCounter(clientB))

	key := "test:cross"
	const window = 60

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); cA.Increment(key, window, 1) }()
		go func() { defer wg.Done(); cB.Increment(key, window, 1) }()
	}
	wg.Wait()

	// Both instances should see the aggregated count.
	gotA := cA.Current(key, window)
	gotB := cB.Current(key, window)
	if gotA != 10 {
		t.Fatalf("instance A: expected 10, got %d", gotA)
	}
	if gotB != 10 {
		t.Fatalf("instance B: expected 10, got %d", gotB)
	}
}

func TestConsumeWorkspaceThroughput_BillingDisabled(t *testing.T) {
	env.BillingEnabled = false
	t.Cleanup(func() { env.BillingEnabled = false })

	wsID := uuid.New()
	if !ConsumeWorkspaceThroughput(&wsID, 1) {
		t.Fatal("expected true when billing is disabled")
	}
}

func TestConsumeWorkspaceThroughput_NilWorkspace(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })

	if !ConsumeWorkspaceThroughput(nil, 1) {
		t.Fatal("expected true when workspace ID is nil")
	}
}

func TestConsumeWorkspaceThroughput_WithinLimit(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })
	resetGlobalCounter()
	t.Cleanup(resetGlobalCounter)

	const limit = 5
	mock := &mockQuery{info: ThroughputInfo{Limit: limit, WindowSeconds: 60}}
	queryThroughputFn = mock.query
	t.Cleanup(func() { queryThroughputFn = queryThroughput })

	wsID := uuid.New()
	InvalidateCache(billing_model.ScopeWorkspace, wsID)

	for i := 0; i < limit; i++ {
		if !ConsumeWorkspaceThroughput(&wsID, 1) {
			t.Fatalf("expected allowed at request %d/%d", i+1, limit)
		}
	}
}

func TestConsumeWorkspaceThroughput_ExceedsLimit(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })
	resetGlobalCounter()
	t.Cleanup(resetGlobalCounter)

	const limit = 3
	mock := &mockQuery{info: ThroughputInfo{Limit: limit, WindowSeconds: 60}}
	queryThroughputFn = mock.query
	t.Cleanup(func() { queryThroughputFn = queryThroughput })

	wsID := uuid.New()
	InvalidateCache(billing_model.ScopeWorkspace, wsID)

	for i := 0; i < limit; i++ {
		ConsumeWorkspaceThroughput(&wsID, 1)
	}

	if ConsumeWorkspaceThroughput(&wsID, 1) {
		t.Fatal("expected false when limit is exceeded")
	}
}

func TestConsumeWorkspaceThroughput_Unlimited(t *testing.T) {
	env.BillingEnabled = true
	t.Cleanup(func() { env.BillingEnabled = false })
	resetGlobalCounter()
	t.Cleanup(resetGlobalCounter)

	mock := &mockQuery{info: ThroughputInfo{Unlimited: true}}
	queryThroughputFn = mock.query
	t.Cleanup(func() { queryThroughputFn = queryThroughput })

	wsID := uuid.New()
	InvalidateCache(billing_model.ScopeWorkspace, wsID)

	for i := 0; i < 1000; i++ {
		if !ConsumeWorkspaceThroughput(&wsID, 1) {
			t.Fatalf("expected allowed at request %d for unlimited workspace", i+1)
		}
	}
}
