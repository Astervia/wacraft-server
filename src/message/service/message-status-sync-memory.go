package message_service

import (
	message_model "github.com/Rfluid/whatsapp-cloud-api/src/message"
	"github.com/google/uuid"
	"time"
)

// MemoryMessageStatusSync wraps MessageStatusSynchronizer so it satisfies
// the MessageStatusSync interface for in-process (single-instance) operation.
type MemoryMessageStatusSync struct {
	inner *MessageStatusSynchronizer
}

func (m *MemoryMessageStatusSync) AddMessage(wamID string, timeout time.Duration) error {
	return m.inner.AddMessage(wamID, timeout)
}

func (m *MemoryMessageStatusSync) MessageSaved(wamID string, messageID uuid.UUID, timeout time.Duration) error {
	return m.inner.MessageSaved(wamID, messageID, timeout)
}

func (m *MemoryMessageStatusSync) RollbackMessage(wamID string, timeout time.Duration) error {
	return m.inner.RollbackMessage(wamID, timeout)
}

func (m *MemoryMessageStatusSync) AddStatus(wamID string, status *message_model.SendingStatus, timeout time.Duration) (uuid.UUID, error) {
	return m.inner.AddStatus(wamID, status, timeout)
}

// CreateMemoryMessageStatusSync returns the default in-memory MessageStatusSync.
func CreateMemoryMessageStatusSync() MessageStatusSync {
	return &MemoryMessageStatusSync{
		inner: CreateMessageStatusSynchronizer(),
	}
}
