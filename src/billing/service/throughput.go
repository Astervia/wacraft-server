package billing_service

import (
	"fmt"
	"time"

	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	synch_contract "github.com/Astervia/wacraft-core/src/synch/contract"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	"github.com/google/uuid"
)

// ThroughputCounter tracks weighted request counts per scope using fixed time windows.
// It wraps a DistributedCounter and embeds the window boundary in each key so that
// the counter resets automatically when the window expires (via TTL).
type ThroughputCounter struct {
	counter synch_contract.DistributedCounter
}

// NewThroughputCounter creates a counter backed by the given DistributedCounter.
func NewThroughputCounter(counter synch_contract.DistributedCounter) *ThroughputCounter {
	return &ThroughputCounter{counter: counter}
}

// Increment adds weight to the counter for the given key and window size.
// Returns the new total count within the current window.
func (c *ThroughputCounter) Increment(key string, windowSeconds int, weight int) int64 {
	bk := bucketKey(key, windowSeconds)
	val, err := c.counter.Increment(bk, int64(weight))
	if err != nil {
		return 0
	}
	if weight > 0 && val == int64(weight) {
		// First increment in this window — set TTL so the key is cleaned up automatically.
		_ = c.counter.SetTTL(bk, time.Duration(windowSeconds)*time.Second)
	}
	return val
}

// Current returns the current count for a key within its window without incrementing.
func (c *ThroughputCounter) Current(key string, windowSeconds int) int64 {
	bk := bucketKey(key, windowSeconds)
	val, _ := c.counter.Get(bk)
	return val
}

// WindowReset returns when the current window expires for the given key.
func (c *ThroughputCounter) WindowReset(key string, windowSeconds int) time.Time {
	if windowSeconds <= 0 {
		return time.Time{}
	}
	bucketStart := time.Now().Unix() / int64(windowSeconds) * int64(windowSeconds)
	return time.Unix(bucketStart+int64(windowSeconds), 0)
}

// bucketKey embeds the current window boundary in the key so counters from
// different windows never collide and expired keys are cleaned up via TTL.
func bucketKey(key string, windowSeconds int) string {
	if windowSeconds <= 0 {
		return key
	}
	bucketStart := time.Now().Unix() / int64(windowSeconds) * int64(windowSeconds)
	return fmt.Sprintf("%s:%d", key, bucketStart)
}

// GlobalCounter is the singleton counter used by the throughput middleware.
// Defaults to an in-memory implementation; replaced via SetThroughputCounter
// when Redis mode is active.
var GlobalCounter = NewThroughputCounter(synch_service.NewMemoryCounter())

// SetThroughputCounter replaces the global counter. Called from src/synch/main.go.
func SetThroughputCounter(c *ThroughputCounter) {
	GlobalCounter = c
}

// Key builds a scope key for the counter.
func Key(scope string, id string) string {
	return fmt.Sprintf("%s:%s", scope, id)
}

// ScopeKeyID returns the UUID string used as counter key for the given scope.
func ScopeKeyID(scope billing_model.Scope, userID *uuid.UUID, workspaceID *uuid.UUID) string {
	if scope == billing_model.ScopeWorkspace && workspaceID != nil {
		return workspaceID.String()
	}
	if userID != nil {
		return userID.String()
	}
	return ""
}
