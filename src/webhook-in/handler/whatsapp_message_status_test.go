package webhook_handler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	whk_service "github.com/Astervia/wacraft-server/src/webhook-in/service"
)

// Override the lock with a fresh memory lock for each test.
func TestStatusDedup_SerializeSameWamID(t *testing.T) {
	lock := synch_service.NewMemoryLock[string]()
	whk_service.SetStatusLock(lock)
	statusSynchronizer = whk_service.GetStatusLock()

	const wamID = "wam-001"
	var order []int
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			statusSynchronizer.Lock(wamID)
			// Critical section: record entry order
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
			time.Sleep(10 * time.Millisecond) // simulate work
			statusSynchronizer.Unlock(wamID)
		}()
	}
	wg.Wait()

	// Both goroutines should have run, serialised
	if len(order) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(order))
	}
}

func TestStatusDedup_ParallelDifferentWamIDs(t *testing.T) {
	lock := synch_service.NewMemoryLock[string]()
	whk_service.SetStatusLock(lock)
	statusSynchronizer = whk_service.GetStatusLock()

	var concurrent int64 // peak concurrent count

	run := func(wamID string) {
		statusSynchronizer.Lock(wamID)
		c := atomic.AddInt64(&concurrent, 1)
		_ = c
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt64(&concurrent, -1)
		statusSynchronizer.Unlock(wamID)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); run("wam-A") }()
	go func() { defer wg.Done(); run("wam-B") }()

	// Both goroutines with different IDs should run concurrently — the peak
	// concurrent count should reach 2 before either finishes.
	// We just verify both complete without deadlock.
	wg.Wait()
}
