package campaign_service

import (
	synch_contract "github.com/Astervia/wacraft-core/src/synch/contract"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
)

// contactLock serialises creation of messaging-product contacts for the same
// phone number, preventing duplicate rows under concurrent campaign sends.
// Defaults to an in-memory lock; replaced at startup by SetContactLock
// when SYNC_BACKEND=redis.
var contactLock synch_contract.DistributedLock[string] = synch_service.NewMemoryLock[string]()

// SetContactLock replaces the active lock implementation. Called once
// during application initialisation by the synch wiring package.
func SetContactLock(l synch_contract.DistributedLock[string]) {
	contactLock = l
}
