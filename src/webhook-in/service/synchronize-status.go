package webhook_service

import (
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
)

// NewStatusSynchronizer initializes a new StatusSynchronizer.
func CreateStatusSynchronizer() *synch_service.MutexSwapper[string] {
	return synch_service.CreateMutexSwapper[string]()
}
