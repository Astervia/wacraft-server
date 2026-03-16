package billing_service

import (
	"encoding/json"
	"time"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	synch_contract "github.com/Astervia/wacraft-core/src/synch/contract"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	"github.com/Astervia/wacraft-server/src/database"
)

const (
	endpointWeightCacheKey = "endpoint-weights"
	endpointWeightCacheTTL = 5 * time.Minute
)

var (
	weightCache synch_contract.DistributedCache        = synch_service.NewMemoryCache()
	weightLock  synch_contract.DistributedLock[string] = synch_service.NewMemoryLock[string]()
)

// SetEndpointWeightCache replaces the weight cache and lock. Called from src/synch/main.go.
func SetEndpointWeightCache(c synch_contract.DistributedCache, l synch_contract.DistributedLock[string]) {
	weightCache = c
	weightLock = l
}

// loadWeightsFn is the function used to load endpoint weights from the database.
// It can be overridden in tests to avoid a real DB connection.
var loadWeightsFn = loadWeightsFromDB

// GetEndpointWeight returns the weight for a given method+path, defaulting to 1.
func GetEndpointWeight(method string, path string) int {
	weights := loadWeights()
	key := method + ":" + path
	if w, exists := weights[key]; exists {
		return w
	}
	return 1
}

// InvalidateEndpointWeightCache forces a reload of endpoint weights on next access.
func InvalidateEndpointWeightCache() {
	_ = weightCache.Delete(endpointWeightCacheKey)
}

func loadWeights() map[string]int {
	// Fast path: cache hit.
	if m, ok := getFromWeightCache(); ok {
		return m
	}

	// Cache miss — lock to prevent multiple concurrent DB loads.
	_ = weightLock.Lock(endpointWeightCacheKey)
	defer weightLock.Unlock(endpointWeightCacheKey) //nolint:errcheck

	// Double-check after acquiring the lock.
	if m, ok := getFromWeightCache(); ok {
		return m
	}

	m := loadWeightsFn()

	if data, err := json.Marshal(m); err == nil {
		_ = weightCache.Set(endpointWeightCacheKey, data, endpointWeightCacheTTL)
	}

	return m
}

func getFromWeightCache() (map[string]int, bool) {
	data, found, err := weightCache.Get(endpointWeightCacheKey)
	if !found || err != nil {
		return nil, false
	}
	var m map[string]int
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false
	}
	return m, true
}

func loadWeightsFromDB() map[string]int {
	var weights []billing_entity.EndpointWeight
	m := make(map[string]int)
	if err := database.DB.Find(&weights).Error; err != nil {
		return m
	}
	for _, w := range weights {
		m[w.Method+":"+w.PathPattern] = w.Weight
	}
	return m
}
