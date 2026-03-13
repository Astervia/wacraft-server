package campaign_service

import (
	"sync"
	"testing"
	"time"

	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
)

func TestContactDedup_SamePhone(t *testing.T) {
	// Reset to a fresh memory lock for this test.
	SetContactLock(synch_service.NewMemoryLock[string]())

	const phone = "+15551234567"
	var order []int
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			contactLock.Lock(phone)
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			contactLock.Unlock(phone)
		}()
	}
	wg.Wait()

	if len(order) != 2 {
		t.Fatalf("expected 2 serialised entries, got %d", len(order))
	}
}

func TestContactDedup_DifferentPhones(t *testing.T) {
	SetContactLock(synch_service.NewMemoryLock[string]())

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan struct{}, 2)

	for _, phone := range []string{"+15551111111", "+15552222222"} {
		phone := phone
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			contactLock.Lock(phone)
			results <- struct{}{}
			// Leave lock held briefly to confirm both acquired their lock concurrently.
			time.Sleep(20 * time.Millisecond)
			contactLock.Unlock(phone)
		}()
	}

	close(start) // release both goroutines simultaneously
	wg.Wait()

	if len(results) != 2 {
		t.Fatalf("expected both goroutines to acquire lock, got %d", len(results))
	}
}
