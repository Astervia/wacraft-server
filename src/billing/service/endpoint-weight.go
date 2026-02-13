package billing_service

import (
	"sync"

	billing_entity "github.com/Astervia/wacraft-core/src/billing/entity"
	"github.com/Astervia/wacraft-server/src/database"
)

// endpointWeightCache caches endpoint weights in memory.
var endpointWeightCache struct {
	mu      sync.RWMutex
	weights map[string]int // key: "METHOD:path" -> weight
	loaded  bool
}

// GetEndpointWeight returns the weight for a given method+path, defaulting to 1.
func GetEndpointWeight(method string, path string) int {
	loadWeightsIfNeeded()

	key := method + ":" + path
	endpointWeightCache.mu.RLock()
	defer endpointWeightCache.mu.RUnlock()

	if w, exists := endpointWeightCache.weights[key]; exists {
		return w
	}
	return 1 // Default weight
}

// InvalidateEndpointWeightCache forces a reload of endpoint weights.
func InvalidateEndpointWeightCache() {
	endpointWeightCache.mu.Lock()
	endpointWeightCache.loaded = false
	endpointWeightCache.mu.Unlock()
}

func loadWeightsIfNeeded() {
	endpointWeightCache.mu.RLock()
	if endpointWeightCache.loaded {
		endpointWeightCache.mu.RUnlock()
		return
	}
	endpointWeightCache.mu.RUnlock()

	endpointWeightCache.mu.Lock()
	defer endpointWeightCache.mu.Unlock()

	// Double-check after acquiring write lock
	if endpointWeightCache.loaded {
		return
	}

	var weights []billing_entity.EndpointWeight
	if err := database.DB.Find(&weights).Error; err != nil {
		endpointWeightCache.weights = make(map[string]int)
		endpointWeightCache.loaded = true
		return
	}

	m := make(map[string]int, len(weights))
	for _, w := range weights {
		key := w.Method + ":" + w.PathPattern
		m[key] = w.Weight
	}
	endpointWeightCache.weights = m
	endpointWeightCache.loaded = true
}
