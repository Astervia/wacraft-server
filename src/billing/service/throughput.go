package billing_service

import (
	"fmt"
	"sync"
	"time"

	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	"github.com/google/uuid"
)

// Counter tracks weighted request counts per scope using fixed time windows.
// Uses MutexSwapper for per-key locking so operations on different keys are fully parallel.
type Counter struct {
	entries sync.Map // map[string]*counterEntry
	swapper *synch_service.MutexSwapper[string]
}

type counterEntry struct {
	count       int64
	windowStart time.Time
	windowSec   int
}

// NewCounter creates a new throughput counter.
func NewCounter() *Counter {
	c := &Counter{
		swapper: synch_service.CreateMutexSwapper[string](),
	}
	go c.cleanup()
	return c
}

// Increment adds weight to the counter for the given key and window size.
// Returns the new total count within the current window.
// Only serializes operations on the same key — different keys run in parallel.
func (c *Counter) Increment(key string, windowSeconds int, weight int) int64 {
	c.swapper.Lock(key)
	defer c.swapper.Unlock(key)

	now := time.Now()
	raw, exists := c.entries.Load(key)

	if !exists {
		entry := &counterEntry{
			count:       int64(weight),
			windowStart: now,
			windowSec:   windowSeconds,
		}
		c.entries.Store(key, entry)
		return entry.count
	}

	entry := raw.(*counterEntry)
	if now.Sub(entry.windowStart).Seconds() >= float64(entry.windowSec) {
		// Window expired, start new one
		entry = &counterEntry{
			count:       int64(weight),
			windowStart: now,
			windowSec:   windowSeconds,
		}
		c.entries.Store(key, entry)
		return entry.count
	}

	entry.count += int64(weight)
	return entry.count
}

// Current returns the current count for a key within its window.
// Lock-free read — does not block other operations.
func (c *Counter) Current(key string) int64 {
	raw, exists := c.entries.Load(key)
	if !exists {
		return 0
	}

	entry := raw.(*counterEntry)
	if time.Since(entry.windowStart).Seconds() >= float64(entry.windowSec) {
		return 0 // Window expired
	}

	return entry.count
}

// WindowReset returns the time when the current window resets for a key.
// Lock-free read.
func (c *Counter) WindowReset(key string) time.Time {
	raw, exists := c.entries.Load(key)
	if !exists {
		return time.Time{}
	}

	entry := raw.(*counterEntry)
	return entry.windowStart.Add(time.Duration(entry.windowSec) * time.Second)
}

// Key builds a scope key for the counter.
func Key(scope string, id string) string {
	return fmt.Sprintf("%s:%s", scope, id)
}

// cleanup periodically removes expired entries.
func (c *Counter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.entries.Range(func(key, value any) bool {
			entry := value.(*counterEntry)
			if now.Sub(entry.windowStart).Seconds() >= float64(entry.windowSec*2) {
				c.entries.Delete(key)
			}
			return true
		})
	}
}

// GlobalCounter is the singleton counter used by the throughput middleware.
var GlobalCounter = NewCounter()

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
