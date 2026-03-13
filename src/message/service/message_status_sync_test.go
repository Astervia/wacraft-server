package message_service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
	message_model "github.com/Rfluid/whatsapp-cloud-api/src/message"
	"github.com/google/uuid"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func dummyStatus() *message_model.SendingStatus {
	st := message_model.SendingStatus("sent")
	return &st
}

// ─── Memory implementation tests ─────────────────────────────────────────────

func TestMemorySync_MessageThenStatus(t *testing.T) {
	s := CreateMemoryMessageStatusSync()
	wamID := uuid.New().String()
	msgID := uuid.New()

	var gotID uuid.UUID
	var statusErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		gotID, statusErr = s.AddStatus(wamID, dummyStatus(), 2*time.Second)
	}()

	// Give AddStatus a moment to register
	time.Sleep(20 * time.Millisecond)

	if err := s.AddMessage(wamID, 2*time.Second); err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if err := s.MessageSaved(wamID, msgID, 2*time.Second); err != nil {
		t.Fatalf("MessageSaved: %v", err)
	}

	wg.Wait()
	if statusErr != nil {
		t.Fatalf("AddStatus: %v", statusErr)
	}
	if gotID != msgID {
		t.Errorf("got ID %v, want %v", gotID, msgID)
	}
}

func TestMemorySync_StatusThenMessage(t *testing.T) {
	s := CreateMemoryMessageStatusSync()
	wamID := uuid.New().String()
	msgID := uuid.New()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.AddMessage(wamID, 2*time.Second)
	}()

	// AddStatus arrives first without waiting for AddMessage to have started
	var gotID uuid.UUID
	var statusErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		gotID, statusErr = s.AddStatus(wamID, dummyStatus(), 2*time.Second)
	}()

	if err := <-errCh; err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if err := s.MessageSaved(wamID, msgID, 2*time.Second); err != nil {
		t.Fatalf("MessageSaved: %v", err)
	}

	wg.Wait()
	if statusErr != nil {
		t.Fatalf("AddStatus: %v", statusErr)
	}
	if gotID != msgID {
		t.Errorf("got ID %v, want %v", gotID, msgID)
	}
}

func TestMemorySync_Timeout(t *testing.T) {
	s := CreateMemoryMessageStatusSync()
	wamID := uuid.New().String()
	err := s.AddMessage(wamID, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestMemorySync_Rollback(t *testing.T) {
	s := CreateMemoryMessageStatusSync()
	wamID := uuid.New().String()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.AddMessage(wamID, 2*time.Second)
	}()

	var gotID uuid.UUID
	var statusErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		gotID, statusErr = s.AddStatus(wamID, dummyStatus(), 2*time.Second)
	}()

	if err := <-errCh; err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if err := s.RollbackMessage(wamID, 2*time.Second); err != nil {
		t.Fatalf("RollbackMessage: %v", err)
	}

	wg.Wait()
	if gotID != uuid.Nil {
		t.Errorf("expected uuid.Nil after rollback, got %v", gotID)
	}
	if statusErr == nil {
		t.Fatal("expected rollback error from AddStatus, got nil")
	}
	if !errors.Is(statusErr, errors.New("message rolled back")) && statusErr.Error() != "message rolled back" {
		t.Errorf("unexpected error: %v", statusErr)
	}
}

func TestMemorySync_ConcurrentDifferentMessages(t *testing.T) {
	const n = 10
	s := CreateMemoryMessageStatusSync()

	type result struct {
		id  uuid.UUID
		err error
	}
	results := make([]result, n)
	wamIDs := make([]string, n)
	msgIDs := make([]uuid.UUID, n)
	for i := 0; i < n; i++ {
		wamIDs[i] = uuid.New().String()
		msgIDs[i] = uuid.New()
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			var inner sync.WaitGroup
			inner.Add(1)
			go func() {
				defer inner.Done()
				results[i].id, results[i].err = s.AddStatus(wamIDs[i], dummyStatus(), 3*time.Second)
			}()
			time.Sleep(5 * time.Millisecond)
			if err := s.AddMessage(wamIDs[i], 3*time.Second); err != nil {
				t.Errorf("AddMessage[%d]: %v", i, err)
				inner.Wait()
				return
			}
			if err := s.MessageSaved(wamIDs[i], msgIDs[i], 3*time.Second); err != nil {
				t.Errorf("MessageSaved[%d]: %v", i, err)
			}
			inner.Wait()
		}()
	}
	wg.Wait()

	for i, r := range results {
		if r.err != nil {
			t.Errorf("AddStatus[%d]: %v", i, r.err)
		}
		if r.id != msgIDs[i] {
			t.Errorf("result[%d]: got %v, want %v", i, r.id, msgIDs[i])
		}
	}
}

// ─── Redis implementation tests ───────────────────────────────────────────────

func testRedisClient(t *testing.T) *synch_redis.Client {
	t.Helper()
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set, skipping Redis integration test")
	}
	client, err := synch_redis.NewClient(synch_redis.Config{
		URL:       redisURL,
		DB:        15,
		KeyPrefix: "wacraft:test:",
	})
	if err != nil {
		t.Fatalf("failed to create Redis client: %v", err)
	}
	ctx := context.Background()
	if err := client.Redis().FlushDB(ctx).Err(); err != nil {
		t.Fatalf("failed to flush test DB: %v", err)
	}
	t.Cleanup(func() {
		client.Redis().FlushDB(ctx)
		client.Close()
	})
	return client
}

func TestRedisSync_CrossInstance(t *testing.T) {
	client := testRedisClient(t)
	syncA := NewRedisMessageStatusSync(client)
	syncB := NewRedisMessageStatusSync(client)
	wamID := uuid.New().String()
	msgID := uuid.New()

	errCh := make(chan error, 1)
	go func() {
		errCh <- syncA.AddMessage(wamID, 5*time.Second)
	}()

	var gotID uuid.UUID
	var statusErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		gotID, statusErr = syncB.AddStatus(wamID, dummyStatus(), 5*time.Second)
	}()

	if err := <-errCh; err != nil {
		t.Fatalf("AddMessage (instance A): %v", err)
	}
	if err := syncA.MessageSaved(wamID, msgID, 5*time.Second); err != nil {
		t.Fatalf("MessageSaved (instance A): %v", err)
	}

	wg.Wait()
	if statusErr != nil {
		t.Fatalf("AddStatus (instance B): %v", statusErr)
	}
	if gotID != msgID {
		t.Errorf("got ID %v, want %v", gotID, msgID)
	}
}

func TestRedisSync_Timeout(t *testing.T) {
	client := testRedisClient(t)
	s := NewRedisMessageStatusSync(client)
	wamID := uuid.New().String()
	err := s.AddMessage(wamID, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestRedisSync_Rollback(t *testing.T) {
	client := testRedisClient(t)
	syncA := NewRedisMessageStatusSync(client)
	syncB := NewRedisMessageStatusSync(client)
	wamID := uuid.New().String()

	errCh := make(chan error, 1)
	go func() {
		errCh <- syncA.AddMessage(wamID, 5*time.Second)
	}()

	var gotID uuid.UUID
	var statusErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		gotID, statusErr = syncB.AddStatus(wamID, dummyStatus(), 5*time.Second)
	}()

	if err := <-errCh; err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if err := syncA.RollbackMessage(wamID, 5*time.Second); err != nil {
		t.Fatalf("RollbackMessage: %v", err)
	}

	wg.Wait()
	if gotID != uuid.Nil {
		t.Errorf("expected uuid.Nil after rollback, got %v", gotID)
	}
	if statusErr == nil || statusErr.Error() != "message rolled back" {
		t.Errorf("expected rollback error, got: %v", statusErr)
	}
}

func TestRedisSync_KeyCleanup(t *testing.T) {
	client := testRedisClient(t)
	syncA := NewRedisMessageStatusSync(client)
	syncB := NewRedisMessageStatusSync(client)
	wamID := uuid.New().String()
	msgID := uuid.New()

	errCh := make(chan error, 1)
	go func() {
		errCh <- syncA.AddMessage(wamID, 5*time.Second)
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		syncB.AddStatus(wamID, dummyStatus(), 5*time.Second)
	}()

	if err := <-errCh; err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if err := syncA.MessageSaved(wamID, msgID, 5*time.Second); err != nil {
		t.Fatalf("MessageSaved: %v", err)
	}
	wg.Wait()

	ctx := context.Background()
	rdb := client.Redis()
	statusKey := fmt.Sprintf("wacraft:test:msg:%s:status", wamID)
	savedKey := fmt.Sprintf("wacraft:test:msg:%s:saved", wamID)

	statusExists, _ := rdb.Exists(ctx, statusKey).Result()
	savedExists, _ := rdb.Exists(ctx, savedKey).Result()

	if statusExists > 0 {
		t.Errorf("status key still exists after handshake: %s", statusKey)
	}
	if savedExists > 0 {
		t.Errorf("saved key still exists after handshake: %s", savedKey)
	}
}
