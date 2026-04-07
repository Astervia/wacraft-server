package campaign_worker

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	"github.com/google/uuid"
)

// runSchedulerWithLock simulates the lock-acquisition step of processCampaign in
// isolation, calling process(id) only when the lock is acquired.
func runSchedulerWithLock(worker *SchedulerWorker, id string, process func(id string)) {
	lockKey := "campaign_schedule:" + id
	if worker.lock != nil {
		acquired, err := worker.lock.TryLock(lockKey)
		if err != nil || !acquired {
			return
		}
		defer worker.lock.Unlock(lockKey) //nolint:errcheck
	}
	process(id)
}

type scheduleCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

func newScheduleCounter() *scheduleCounter {
	return &scheduleCounter{counts: make(map[string]int)}
}

func (sc *scheduleCounter) record(id string) {
	sc.mu.Lock()
	sc.counts[id]++
	sc.mu.Unlock()
}

func (sc *scheduleCounter) get(id string) int {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.counts[id]
}

// TestSchedulerWorker_MemoryMode verifies that all campaigns are processed when
// no distributed lock is configured (memory / single-instance mode).
func TestSchedulerWorker_MemoryMode(t *testing.T) {
	worker := &SchedulerWorker{lock: nil}
	counter := newScheduleCounter()

	campaignIDs := make([]string, 5)
	for i := range campaignIDs {
		campaignIDs[i] = uuid.New().String()
	}

	for _, id := range campaignIDs {
		runSchedulerWithLock(worker, id, counter.record)
	}

	for _, id := range campaignIDs {
		if n := counter.get(id); n != 1 {
			t.Errorf("campaign %s: expected 1 process call, got %d", id, n)
		}
	}
}

// TestSchedulerWorker_NoDuplicateExecution_RedisMode verifies that two workers
// sharing a lock backend never execute the same campaign concurrently.
func TestSchedulerWorker_NoDuplicateExecution_RedisMode(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set — skipping Redis integration test")
	}

	clientA, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 14, LockTTL: 5 * time.Second})
	if err != nil {
		t.Fatalf("clientA: %v", err)
	}
	clientB, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 14, LockTTL: 5 * time.Second})
	if err != nil {
		t.Fatalf("clientB: %v", err)
	}

	clientA.Redis().FlushDB(t.Context())
	t.Cleanup(func() { clientA.Redis().FlushDB(t.Context()) })

	workerA := &SchedulerWorker{lock: synch_redis.NewRedisLock[string](clientA)}
	workerB := &SchedulerWorker{lock: synch_redis.NewRedisLock[string](clientB)}

	const numCampaigns = 8
	campaignIDs := make([]string, numCampaigns)
	for i := range campaignIDs {
		campaignIDs[i] = uuid.New().String()
	}

	var trackMu sync.Mutex
	current := make(map[string]int)
	peak := make(map[string]int)

	processWithHold := func(id string) {
		trackMu.Lock()
		current[id]++
		if current[id] > peak[id] {
			peak[id] = current[id]
		}
		trackMu.Unlock()

		time.Sleep(30 * time.Millisecond)

		trackMu.Lock()
		current[id]--
		trackMu.Unlock()
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, id := range campaignIDs {
		id := id
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			runSchedulerWithLock(workerA, id, processWithHold)
		}()
		go func() {
			defer wg.Done()
			<-start
			runSchedulerWithLock(workerB, id, processWithHold)
		}()
	}
	close(start)
	wg.Wait()

	for _, id := range campaignIDs {
		if n := peak[id]; n > 1 {
			t.Errorf("campaign %s: %d concurrent executions, want ≤1", id, n)
		}
	}
}

// TestSchedulerWorker_LockExpiry verifies that after a lock's TTL expires,
// another worker can acquire it and process the campaign.
func TestSchedulerWorker_LockExpiry(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set — skipping Redis integration test")
	}

	clientA, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 14, LockTTL: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("clientA: %v", err)
	}
	clientB, err := synch_redis.NewClient(synch_redis.Config{URL: redisURL, DB: 14, LockTTL: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("clientB: %v", err)
	}

	clientA.Redis().FlushDB(t.Context())
	t.Cleanup(func() { clientA.Redis().FlushDB(t.Context()) })

	workerA := &SchedulerWorker{lock: synch_redis.NewRedisLock[string](clientA)}
	workerB := &SchedulerWorker{lock: synch_redis.NewRedisLock[string](clientB)}

	campaignID := uuid.New().String()
	counter := newScheduleCounter()

	lockKey := "campaign_schedule:" + campaignID
	acquired, err := workerA.lock.TryLock(lockKey)
	if err != nil || !acquired {
		t.Fatalf("workerA failed to acquire lock")
	}
	counter.record(campaignID)

	// workerB tries immediately — should be skipped.
	runSchedulerWithLock(workerB, campaignID, counter.record)
	if n := counter.get(campaignID); n != 1 {
		t.Fatalf("before expiry: expected 1 process, got %d", n)
	}

	// Wait for TTL to expire.
	time.Sleep(300 * time.Millisecond)

	// workerB retries — lock has expired, should process.
	runSchedulerWithLock(workerB, campaignID, counter.record)
	if n := counter.get(campaignID); n != 2 {
		t.Fatalf("after expiry: expected 2 processes, got %d", n)
	}
}

// TestSchedulerWorker_GracefulShutdown verifies that Stop() waits for in-flight
// processing to complete before returning.
func TestSchedulerWorker_GracefulShutdown(t *testing.T) {
	worker := NewSchedulerWorker()

	var inFlight atomic.Int32
	var completed atomic.Int32

	worker.wg.Add(1)
	go func() {
		defer worker.wg.Done()
		inFlight.Store(1)
		select {
		case <-worker.ctx.Done():
		case <-time.After(2 * time.Second):
		}
		inFlight.Store(0)
		completed.Store(1)
	}()

	for inFlight.Load() == 0 {
		time.Sleep(time.Millisecond)
	}

	worker.Stop()

	if completed.Load() != 1 {
		t.Fatal("Stop returned before in-flight work completed")
	}
}

// TestSchedulerWorker_ProcessSkipsIfLocked verifies that a campaign is skipped
// when another worker already holds the lock.
func TestSchedulerWorker_ProcessSkipsIfLocked(t *testing.T) {
	lock := synch_service.NewMemoryLock[string]()
	workerA := &SchedulerWorker{lock: lock}
	workerB := &SchedulerWorker{lock: lock}

	campaignID := uuid.New().String()

	lockKey := "campaign_schedule:" + campaignID
	acquired, err := workerA.lock.TryLock(lockKey)
	if err != nil || !acquired {
		t.Fatalf("workerA failed to acquire lock")
	}
	defer workerA.lock.Unlock(lockKey) //nolint:errcheck

	counter := newScheduleCounter()
	runSchedulerWithLock(workerB, campaignID, counter.record)

	if n := counter.get(campaignID); n != 0 {
		t.Fatalf("workerB should have skipped, but recorded %d process calls", n)
	}
}

// TestSchedulerWorker_NilLockProcessesAll verifies that with no lock (memory mode)
// a worker processes all campaigns.
func TestSchedulerWorker_NilLockProcessesAll(t *testing.T) {
	worker := &SchedulerWorker{lock: nil}

	const n = 5
	ids := make([]string, n)
	for i := range ids {
		ids[i] = uuid.New().String()
	}

	var total atomic.Int64
	var wg sync.WaitGroup
	for _, id := range ids {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			runSchedulerWithLock(worker, id, func(_ string) { total.Add(1) })
		}()
	}
	wg.Wait()

	if got := total.Load(); got != int64(n) {
		t.Fatalf("expected %d total, got %d", n, got)
	}
}

// TestSchedulerWorker_PollIntervalDefault verifies the default poll interval.
func TestSchedulerWorker_PollIntervalDefault(t *testing.T) {
	d := schedulerPollInterval()
	if d != 30*time.Second {
		t.Errorf("default poll interval: got %v, want 30s", d)
	}
}

// TestSchedulerWorker_PollIntervalFromEnv verifies that CAMPAIGN_SCHEDULE_POLL_INTERVAL
// is respected.
func TestSchedulerWorker_PollIntervalFromEnv(t *testing.T) {
	t.Setenv("CAMPAIGN_SCHEDULE_POLL_INTERVAL", "1m")
	d := schedulerPollInterval()
	if d != time.Minute {
		t.Errorf("env poll interval: got %v, want 1m", d)
	}
}
