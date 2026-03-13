package webhook_service

import (
	synch_contract "github.com/Astervia/wacraft-core/src/synch/contract"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
)

// statusLock serialises processing of concurrent status updates for the same
// wamID. Defaults to an in-memory lock; replaced at startup by SetStatusLock
// when SYNC_BACKEND=redis.
var statusLock synch_contract.DistributedLock[string] = synch_service.NewMemoryLock[string]()

// SetStatusLock replaces the active lock implementation. Called once during
// application initialisation by the synch wiring package.
func SetStatusLock(l synch_contract.DistributedLock[string]) {
	statusLock = l
}

// GetStatusLock returns the active distributed lock for use by handlers.
func GetStatusLock() synch_contract.DistributedLock[string] {
	return statusLock
}
