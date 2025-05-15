package campaign_service

import synch_service "github.com/Astervia/wacraft-core/src/synch/service"

func CreateContactSynchronizer() *synch_service.MutexSwapper[string] {
	return synch_service.CreateMutexSwapper[string]()
}
