package billing_service

import (
	"os"
	"sync"
	"testing"
	"time"

	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
)

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
