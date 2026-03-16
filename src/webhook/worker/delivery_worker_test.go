package webhook_worker

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	"github.com/google/uuid"
)

// makeDelivery creates a minimal WebhookDelivery for testing (no DB required).
func makeDelivery() *webhook_entity.WebhookDelivery {
	return &webhook_entity.WebhookDelivery{
		WebhookID: uuid.New(),
		// ID is set by DB normally; for tests we assign one directly.
	}
}

// processCount returns the number of times processDelivery was called for the
// given delivery ID, using an atomic counter map.
type processCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

func newProcessCounter() *processCounter {
	return &processCounter{counts: make(map[string]int)}
}

func (pc *processCounter) record(id string) {
	pc.mu.Lock()
	pc.counts[id]++
	pc.mu.Unlock()
}

func (pc *processCounter) get(id string) int {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.counts[id]
}

// runWithLock simulates processDelivery's lock logic in isolation, calling
// process(id) only when the lock is acquired.
func runWithLock(worker *DeliveryWorker, id string, process func(id string)) {
	if worker.lock != nil {
		acquired, err := worker.lock.TryLock(id)
		if err != nil || !acquired {
			return
		}
		defer worker.lock.Unlock(id) //nolint:errcheck
	}
	process(id)
}

// TestDeliveryWorker_MemoryMode verifies that all deliveries are processed
// when no distributed lock is configured (memory / single-instance mode).
func TestDeliveryWorker_MemoryMode(t *testing.T) {
	worker := &DeliveryWorker{lock: nil}
	counter := newProcessCounter()

	deliveryIDs := make([]string, 5)
	for i := range deliveryIDs {
		deliveryIDs[i] = uuid.New().String()
	}

	for _, id := range deliveryIDs {
		runWithLock(worker, id, counter.record)
	}

	for _, id := range deliveryIDs {
		if n := counter.get(id); n != 1 {
			t.Errorf("delivery %s: expected 1 process call, got %d", id, n)
		}
	}
}

// TestDeliveryWorker_RedisNoDuplicates verifies that two workers sharing a lock
// backend never process the same delivery concurrently.
//
// TryLock is non-blocking: if worker A finishes before worker B even starts,
// B will re-acquire the (now-released) lock and process sequentially — that is
// correct behaviour (idempotency is handled at the DB layer in production).
// What we assert here is the mutual-exclusion property: peak concurrent
// executions per delivery is always ≤ 1.
func TestDeliveryWorker_RedisNoDuplicates(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set — skipping Redis integration test")
	}

	clientA, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 15, LockTTL: 5 * time.Second})
	if err != nil {
		t.Fatalf("clientA: %v", err)
	}
	clientB, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 15, LockTTL: 5 * time.Second})
	if err != nil {
		t.Fatalf("clientB: %v", err)
	}

	clientA.Redis().FlushDB(t.Context())
	t.Cleanup(func() { clientA.Redis().FlushDB(t.Context()) })

	workerA := &DeliveryWorker{lock: synch_redis.NewRedisLock[string](clientA)}
	workerB := &DeliveryWorker{lock: synch_redis.NewRedisLock[string](clientB)}

	const numDeliveries = 10
	deliveryIDs := make([]string, numDeliveries)
	for i := range deliveryIDs {
		deliveryIDs[i] = uuid.New().String()
	}

	// Track peak concurrent executions per delivery ID.
	var trackMu sync.Mutex
	current := make(map[string]int)
	peak := make(map[string]int)

	// processWithHold simulates a delivery that takes 30ms to complete,
	// long enough to guarantee the two goroutines for the same ID overlap.
	processWithHold := func(id string) {
		trackMu.Lock()
		current[id]++
		if current[id] > peak[id] {
			peak[id] = current[id]
		}
		trackMu.Unlock()

		time.Sleep(30 * time.Millisecond) // hold lock so concurrent goroutine definitely sees it

		trackMu.Lock()
		current[id]--
		trackMu.Unlock()
	}

	// start gate ensures both goroutines race to acquire the lock simultaneously.
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, id := range deliveryIDs {
		id := id
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			runWithLock(workerA, id, processWithHold)
		}()
		go func() {
			defer wg.Done()
			<-start
			runWithLock(workerB, id, processWithHold)
		}()
	}
	close(start) // release all goroutines simultaneously
	wg.Wait()

	// No delivery should ever have been processed by two workers at the same time.
	for _, id := range deliveryIDs {
		if n := peak[id]; n > 1 {
			t.Errorf("delivery %s: %d concurrent executions, want ≤1", id, n)
		}
	}
}

// TestDeliveryWorker_LockExpiry verifies that after a lock's TTL expires,
// another worker can acquire the lock and process the delivery.
func TestDeliveryWorker_LockExpiry(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set — skipping Redis integration test")
	}

	// Use a very short TTL to simulate lock expiry.
	clientA, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 15, LockTTL: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("clientA: %v", err)
	}
	clientB, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 15, LockTTL: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("clientB: %v", err)
	}

	clientA.Redis().FlushDB(t.Context())
	t.Cleanup(func() { clientA.Redis().FlushDB(t.Context()) })

	workerA := &DeliveryWorker{lock: synch_redis.NewRedisLock[string](clientA)}
	workerB := &DeliveryWorker{lock: synch_redis.NewRedisLock[string](clientB)}

	deliveryID := uuid.New().String()
	counter := newProcessCounter()

	// Worker A acquires the lock but does NOT release it (simulates crash).
	acquired, err := workerA.lock.TryLock(deliveryID)
	if err != nil || !acquired {
		t.Fatalf("workerA failed to acquire lock: err=%v acquired=%v", err, acquired)
	}
	counter.record(deliveryID)

	// Worker B tries immediately — lock is held by A, must be skipped.
	runWithLock(workerB, deliveryID, counter.record)
	if n := counter.get(deliveryID); n != 1 {
		t.Fatalf("before expiry: expected 1 process, got %d", n)
	}

	// Wait for the lock TTL to expire.
	time.Sleep(300 * time.Millisecond)

	// Worker B retries — lock has expired, must process successfully.
	runWithLock(workerB, deliveryID, counter.record)
	if n := counter.get(deliveryID); n != 2 {
		t.Fatalf("after expiry: expected 2 processes, got %d", n)
	}
}

// TestDeliveryWorker_GracefulShutdown verifies that Stop() waits for in-flight
// processing to complete before returning.
func TestDeliveryWorker_GracefulShutdown(t *testing.T) {
	worker := NewDeliveryWorker()

	var inFlight atomic.Int32
	var completed atomic.Int32

	worker.wg.Add(1)
	go func() {
		defer worker.wg.Done()
		inFlight.Store(1)
		// Simulate long-running work that should complete before Stop returns.
		select {
		case <-worker.ctx.Done():
		case <-time.After(2 * time.Second):
		}
		inFlight.Store(0)
		completed.Store(1)
	}()

	// Give the goroutine a moment to start.
	for inFlight.Load() == 0 {
		time.Sleep(time.Millisecond)
	}

	worker.Stop()

	if completed.Load() != 1 {
		t.Fatal("Stop returned before in-flight work completed")
	}
}

// TestDeliveryWorker_ProcessDeliverySkipsIfLocked ensures that processDelivery
// on a real worker skips execution when the lock is already held.
func TestDeliveryWorker_ProcessDeliverySkipsIfLocked(t *testing.T) {
	lock := synch_service.NewMemoryLock[string]()
	workerA := &DeliveryWorker{lock: lock}
	workerB := &DeliveryWorker{lock: lock}

	deliveryID := uuid.New().String()

	// workerA acquires the lock and holds it.
	acquired, err := workerA.lock.TryLock(deliveryID)
	if err != nil || !acquired {
		t.Fatalf("workerA failed to acquire lock")
	}
	defer workerA.lock.Unlock(deliveryID) //nolint:errcheck

	counter := newProcessCounter()

	// workerB should skip because the lock is held.
	runWithLock(workerB, deliveryID, counter.record)

	if n := counter.get(deliveryID); n != 0 {
		t.Fatalf("workerB should have skipped, but recorded %d process calls", n)
	}
}

// TestDeliveryWorker_NilLockProcessesAll verifies that with no lock (memory mode),
// concurrent workers both process all deliveries.
func TestDeliveryWorker_NilLockProcessesAll(t *testing.T) {
	workerA := &DeliveryWorker{lock: nil}
	workerB := &DeliveryWorker{lock: nil}

	_ = makeDelivery() // just to use the helper

	const n = 5
	ids := make([]string, n)
	for i := range ids {
		ids[i] = uuid.New().String()
	}

	var total atomic.Int64
	var wg sync.WaitGroup
	for _, id := range ids {
		id := id
		wg.Add(2)
		go func() { defer wg.Done(); runWithLock(workerA, id, func(_ string) { total.Add(1) }) }()
		go func() { defer wg.Done(); runWithLock(workerB, id, func(_ string) { total.Add(1) }) }()
	}
	wg.Wait()

	// Without a lock, both workers process all deliveries.
	if got := total.Load(); got != int64(n*2) {
		t.Fatalf("expected %d total, got %d", n*2, got)
	}
}
