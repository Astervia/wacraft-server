package billing_service

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
)

// resetWeightCache resets the global weight cache/lock to a fresh in-memory implementation.
func resetWeightCache() {
	weightCache = synch_service.NewMemoryCache()
	weightLock = synch_service.NewMemoryLock[string]()
}

func TestEndpointWeightCache_LazyLoad(t *testing.T) {
	resetWeightCache()

	var calls atomic.Int64
	loadWeightsFn = func() map[string]int {
		calls.Add(1)
		return map[string]int{"GET:/api/test": 3}
	}
	t.Cleanup(func() { loadWeightsFn = loadWeightsFromDB })

	// First call triggers DB load.
	w := GetEndpointWeight("GET", "/api/test")
	if w != 3 {
		t.Fatalf("expected 3, got %d", w)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 load, got %d", calls.Load())
	}

	// Second call uses cached value.
	GetEndpointWeight("GET", "/api/test")
	if calls.Load() != 1 {
		t.Fatalf("expected no extra load, got %d", calls.Load())
	}
}

func TestEndpointWeightCache_Invalidate(t *testing.T) {
	resetWeightCache()

	var calls atomic.Int64
	loadWeightsFn = func() map[string]int {
		calls.Add(1)
		return map[string]int{"POST:/msg": 2}
	}
	t.Cleanup(func() { loadWeightsFn = loadWeightsFromDB })

	GetEndpointWeight("POST", "/msg")
	if calls.Load() != 1 {
		t.Fatalf("expected 1 load, got %d", calls.Load())
	}

	InvalidateEndpointWeightCache()

	GetEndpointWeight("POST", "/msg")
	if calls.Load() != 2 {
		t.Fatalf("expected 2 loads after invalidation, got %d", calls.Load())
	}
}

func TestEndpointWeightCache_CrossInstance(t *testing.T) {
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
	t.Cleanup(func() {
		clientA.Redis().FlushDB(t.Context())
		resetWeightCache()
		loadWeightsFn = loadWeightsFromDB
	})

	var calls atomic.Int64
	loadWeightsFn = func() map[string]int {
		calls.Add(1)
		return map[string]int{"DELETE:/resource": 5}
	}

	cacheA := synch_redis.NewRedisCache(clientA)
	lockA := synch_redis.NewRedisLock[string](clientA)
	cacheB := synch_redis.NewRedisCache(clientB)
	lockB := synch_redis.NewRedisLock[string](clientB)

	// Instance A loads and caches.
	SetEndpointWeightCache(cacheA, lockA)
	if w := GetEndpointWeight("DELETE", "/resource"); w != 5 {
		t.Fatalf("instance A: expected 5, got %d", w)
	}

	// Invalidate from instance A.
	InvalidateEndpointWeightCache()

	// Instance B reads after invalidation — must reload from DB.
	SetEndpointWeightCache(cacheB, lockB)
	if w := GetEndpointWeight("DELETE", "/resource"); w != 5 {
		t.Fatalf("instance B after invalidation: expected 5, got %d", w)
	}

	if calls.Load() != 2 {
		t.Fatalf("expected 2 DB loads (initial + after invalidation), got %d", calls.Load())
	}
}

func TestEndpointWeightCache_DefaultWeight(t *testing.T) {
	resetWeightCache()
	loadWeightsFn = func() map[string]int { return map[string]int{} }
	t.Cleanup(func() { loadWeightsFn = loadWeightsFromDB })

	if w := GetEndpointWeight("GET", "/unknown"); w != 1 {
		t.Fatalf("expected default weight 1, got %d", w)
	}
}

func TestEndpointWeightCache_NoConcurrentLoads(t *testing.T) {
	resetWeightCache()

	var calls atomic.Int64
	loadWeightsFn = func() map[string]int {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond)
		return map[string]int{}
	}
	t.Cleanup(func() { loadWeightsFn = loadWeightsFromDB })

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			GetEndpointWeight("GET", "/any")
		}()
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 DB load, got %d", got)
	}
}
