package campaign_service

import (
	"context"
	"os"
	"testing"

	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	synch "github.com/Astervia/wacraft-core/src/synch"
	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
)

// testFactory returns a factory wired to a test Redis DB (or memory if
// REDIS_URL is not set — Redis-specific tests are skipped in that case).
func testRedisClientCampaign(t *testing.T) *synch_redis.Client {
	t.Helper()
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set, skipping Redis integration test")
	}
	client, err := synch_redis.NewClient(synch_redis.Config{
		URL:       redisURL,
		DB:        12,
		KeyPrefix: "wacraft:test:",
	})
	if err != nil {
		t.Fatalf("create Redis client: %v", err)
	}
	client.Redis().FlushDB(context.Background())
	t.Cleanup(func() {
		client.Redis().FlushDB(context.Background())
		client.Close()
	})
	return client
}

// ─── Memory counter tests ─────────────────────────────────────────────────────

func TestCampaignResults_MemoryCounters(t *testing.T) {
	f := synch.NewFactory(synch.BackendMemory, nil)
	counter := f.NewCounter()

	results := campaign_model.CreateCampaignResultsWithCounter(3, counter, "test-campaign")

	results.HandleError(nil, func(r *campaign_model.CampaignResults) {})      // success
	results.HandleError(nil, func(r *campaign_model.CampaignResults) {})      // success
	results.HandleError(errDummy, func(r *campaign_model.CampaignResults) {}) // error

	if results.Sent != 3 {
		t.Errorf("Sent: got %d, want 3", results.Sent)
	}
	if results.Successes != 2 {
		t.Errorf("Successes: got %d, want 2", results.Successes)
	}
	if results.Errors != 1 {
		t.Errorf("Errors: got %d, want 1", results.Errors)
	}
}

// ─── Redis counter tests ──────────────────────────────────────────────────────

func TestCampaignResults_RedisCounters(t *testing.T) {
	client := testRedisClientCampaign(t)
	fA := synch.NewFactory(synch.BackendRedis, client)
	fB := synch.NewFactory(synch.BackendRedis, client)

	counterA := fA.NewCounter()
	counterB := fB.NewCounter()

	campaignID := "redis-campaign-001"
	resultsA := campaign_model.CreateCampaignResultsWithCounter(4, counterA, campaignID)
	resultsB := campaign_model.CreateCampaignResultsWithCounter(4, counterB, campaignID)

	// Instance A: 2 successes
	resultsA.HandleError(nil, func(*campaign_model.CampaignResults) {})
	resultsA.HandleError(nil, func(*campaign_model.CampaignResults) {})

	// Instance B: 1 success + 1 error
	resultsB.HandleError(nil, func(*campaign_model.CampaignResults) {})
	resultsB.HandleError(errDummy, func(*campaign_model.CampaignResults) {})

	// Read from either instance
	sentA, _ := counterA.Get("sent:" + campaignID)
	succA, _ := counterA.Get("successes:" + campaignID)
	errA, _ := counterA.Get("errors:" + campaignID)

	if sentA != 4 {
		t.Errorf("sent: got %d, want 4", sentA)
	}
	if succA != 3 {
		t.Errorf("successes: got %d, want 3", succA)
	}
	if errA != 1 {
		t.Errorf("errors: got %d, want 1", errA)
	}
}

// errDummy is a sentinel error for testing HandleError.
type dummyError struct{}

func (dummyError) Error() string { return "dummy error" }

var errDummy = dummyError{}
