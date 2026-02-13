package billing_service

import (
	"sync"
	"time"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	billing_model "github.com/Astervia/wacraft-core/src/billing/model"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/google/uuid"
)

// ThroughputInfo holds the resolved throughput limit and window for a scope.
// When Unlimited is true, no counting or enforcement is performed.
type ThroughputInfo struct {
	Limit         int
	WindowSeconds int
	Unlimited     bool
}

// subscriptionCache caches active subscriptions per scope key with TTL.
// Uses sync.Map for lock-free reads and MutexSwapper for per-key write serialization
// to prevent thundering herd on the same key without blocking unrelated keys.
type subscriptionCache struct {
	entries sync.Map // map[string]*cacheEntry
	swapper *synch_service.MutexSwapper[string]
	ttl     time.Duration
}

type cacheEntry struct {
	info      ThroughputInfo
	fetchedAt time.Time
}

var cache = &subscriptionCache{
	swapper: synch_service.CreateMutexSwapper[string](),
	ttl:     5 * time.Minute,
}

// ResolveThroughput returns the effective throughput limit for a given scope.
// It sums all active subscription limits. Falls back to the default free plan if none.
func ResolveThroughput(scope billing_model.Scope, userID *uuid.UUID, workspaceID *uuid.UUID) ThroughputInfo {
	var key string
	if scope == billing_model.ScopeUser && userID != nil {
		key = Key(string(scope), userID.String())
	} else if scope == billing_model.ScopeWorkspace && workspaceID != nil {
		key = Key(string(scope), workspaceID.String())
	} else {
		return DefaultFreeInfo()
	}

	// Lock-free cache read
	if raw, exists := cache.entries.Load(key); exists {
		entry := raw.(*cacheEntry)
		if time.Since(entry.fetchedAt) < cache.ttl {
			return entry.info
		}
	}

	// Cache miss or stale — lock only this key to prevent thundering herd
	cache.swapper.Lock(key)
	defer cache.swapper.Unlock(key)

	// Double-check after acquiring per-key lock (another goroutine may have populated it)
	if raw, exists := cache.entries.Load(key); exists {
		entry := raw.(*cacheEntry)
		if time.Since(entry.fetchedAt) < cache.ttl {
			return entry.info
		}
	}

	// Query active subscriptions from database
	info := queryThroughput(scope, userID, workspaceID)

	// Update cache
	cache.entries.Store(key, &cacheEntry{
		info:      info,
		fetchedAt: time.Now(),
	})

	return info
}

// InvalidateCache removes a specific scope key from the cache.
func InvalidateCache(scope billing_model.Scope, id uuid.UUID) {
	key := Key(string(scope), id.String())
	cache.entries.Delete(key)
}

func queryThroughput(scope billing_model.Scope, userID *uuid.UUID, workspaceID *uuid.UUID) ThroughputInfo {
	now := time.Now()
	var subscriptions []billing_entity.Subscription

	query := database.DB.
		Preload("Plan").
		Where("scope = ? AND starts_at <= ? AND expires_at > ? AND cancelled_at IS NULL", scope, now, now)

	if scope == billing_model.ScopeUser && userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else if scope == billing_model.ScopeWorkspace && workspaceID != nil {
		query = query.Where("workspace_id = ?", *workspaceID)
	} else {
		return DefaultFreeInfo()
	}

	if err := query.Find(&subscriptions).Error; err != nil || len(subscriptions) == 0 {
		return DefaultFreeInfo()
	}

	// Sum throughputs from all active subscriptions, use the smallest window.
	// If any subscription has limit <= 0, the scope gets unlimited throughput.
	totalLimit := 0
	windowSeconds := 0

	for _, sub := range subscriptions {
		if sub.Plan == nil {
			continue
		}
		effective := sub.EffectiveThroughput(sub.Plan.ThroughputLimit)
		if effective <= 0 {
			// A plan with limit <= 0 means unlimited throughput
			return ThroughputInfo{Unlimited: true, WindowSeconds: sub.Plan.WindowSeconds}
		}
		totalLimit += effective
		if windowSeconds == 0 || sub.Plan.WindowSeconds < windowSeconds {
			windowSeconds = sub.Plan.WindowSeconds
		}
	}

	if totalLimit == 0 {
		return DefaultFreeInfo()
	}

	return ThroughputInfo{
		Limit:         totalLimit,
		WindowSeconds: windowSeconds,
	}
}

// GetDefaultFreePlan returns the default free plan from the database, or nil.
func GetDefaultFreePlan() *billing_entity.Plan {
	var plan billing_entity.Plan
	if err := database.DB.Where("is_default = ? AND active = ?", true, true).First(&plan).Error; err != nil {
		return nil
	}
	return &plan
}

func DefaultFreeInfo() ThroughputInfo {
	// Try DB first
	plan := GetDefaultFreePlan()
	if plan != nil {
		return ThroughputInfo{
			Limit:         plan.ThroughputLimit,
			WindowSeconds: plan.WindowSeconds,
		}
	}
	// Fall back to env defaults
	return ThroughputInfo{
		Limit:         env.DefaultFreeThroughput,
		WindowSeconds: env.DefaultFreeWindow,
	}
}
