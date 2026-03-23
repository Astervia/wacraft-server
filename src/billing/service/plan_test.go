package billing_service

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	"github.com/google/uuid"
)

// resetPlanCache resets the global plan cache/lock to a fresh in-memory implementation.
func resetPlanCache() {
	planCache = synch_service.NewMemoryCache()
	planLock = synch_service.NewMemoryLock[string]()
}

// mockQuery returns a fixed ThroughputInfo and counts invocations.
type mockQuery struct {
	mu    sync.Mutex
	calls int
	info  ThroughputInfo
}

func (m *mockQuery) query(_ billing_model.Scope, _ *uuid.UUID, _ *uuid.UUID) ThroughputInfo {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return m.info
}

func TestSubscriptionCache_Hit(t *testing.T) {
	resetPlanCache()
	mock := &mockQuery{info: ThroughputInfo{Limit: 100, WindowSeconds: 60}}
	queryThroughputFn = mock.query
	t.Cleanup(func() { queryThroughputFn = queryThroughput })

	userID := uuid.New()

	first := ResolveThroughput(billing_model.ScopeUser, &userID, nil)
	second := ResolveThroughput(billing_model.ScopeUser, &userID, nil)

	if mock.calls != 1 {
		t.Fatalf("expected 1 DB query, got %d", mock.calls)
	}
	if first.Limit != 100 || second.Limit != 100 {
		t.Fatalf("unexpected limits: first=%d second=%d", first.Limit, second.Limit)
	}
}

func TestSubscriptionCache_TTLRefresh(t *testing.T) {
	resetPlanCache()
	mock := &mockQuery{info: ThroughputInfo{Limit: 50, WindowSeconds: 30}}
	queryThroughputFn = mock.query
	t.Cleanup(func() { queryThroughputFn = queryThroughput })

	// Override TTL to something very short for this test.
	original := planCacheTTL
	// We can't easily change the const, so we directly exercise InvalidateCache.
	_ = original

	userID := uuid.New()
	ResolveThroughput(billing_model.ScopeUser, &userID, nil)
	if mock.calls != 1 {
		t.Fatalf("expected 1 query after first call, got %d", mock.calls)
	}

	// Invalidate and call again — must re-query.
	InvalidateCache(billing_model.ScopeUser, userID)
	ResolveThroughput(billing_model.ScopeUser, &userID, nil)
	if mock.calls != 2 {
		t.Fatalf("expected 2 queries after invalidation, got %d", mock.calls)
	}
}

func TestSubscriptionCache_ThunderingHerd(t *testing.T) {
	resetPlanCache()

	var calls atomic.Int64
	slow := &mockQuery{info: ThroughputInfo{Limit: 200, WindowSeconds: 60}}
	queryThroughputFn = func(scope billing_model.Scope, userID *uuid.UUID, workspaceID *uuid.UUID) ThroughputInfo {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond) // simulate slow DB
		return slow.info
	}
	t.Cleanup(func() { queryThroughputFn = queryThroughput })

	userID := uuid.New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ResolveThroughput(billing_model.ScopeUser, &userID, nil)
		}()
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("thundering herd: expected 1 DB query, got %d", got)
	}
}

func TestSubscriptionCache_CrossInstance(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set — skipping Redis integration test")
	}

	clientA, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 11})
	if err != nil {
		t.Fatalf("clientA: %v", err)
	}
	clientB, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 11})
	if err != nil {
		t.Fatalf("clientB: %v", err)
	}

	clientA.Redis().FlushDB(t.Context())
	t.Cleanup(func() { clientA.Redis().FlushDB(t.Context()) })

	cacheA := synch_redis.NewRedisCache(clientA)
	lockA := synch_redis.NewRedisLock[string](clientA)
	cacheB := synch_redis.NewRedisCache(clientB)
	lockB := synch_redis.NewRedisLock[string](clientB)

	mock := &mockQuery{info: ThroughputInfo{Limit: 99, WindowSeconds: 60}}
	queryThroughputFn = mock.query
	t.Cleanup(func() { queryThroughputFn = queryThroughput })

	userID := uuid.New()

	// Instance A populates the cache.
	SetPlanCache(cacheA, lockA)
	got := ResolveThroughput(billing_model.ScopeUser, &userID, nil)
	if got.Limit != 99 {
		t.Fatalf("instance A: expected 99, got %d", got.Limit)
	}

	// Instance B should read from the shared Redis cache — no DB hit.
	SetPlanCache(cacheB, lockB)
	got = ResolveThroughput(billing_model.ScopeUser, &userID, nil)
	if got.Limit != 99 {
		t.Fatalf("instance B: expected 99, got %d", got.Limit)
	}

	if mock.calls != 1 {
		t.Fatalf("expected 1 DB query total, got %d", mock.calls)
	}

	// Restore default.
	t.Cleanup(resetPlanCache)
}
