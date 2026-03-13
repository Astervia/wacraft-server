package message_service

import (
	"time"

	message_model "github.com/Rfluid/whatsapp-cloud-api/src/message"
	"github.com/google/uuid"
)

// MessageStatusSync coordinates the handshake between a sent message and its
// first incoming status update. Both sides block until the other arrives.
//
// In memory mode the implementation uses Go channels (single-instance).
// In Redis mode the implementation uses BLPOP/RPUSH lists as single-use
// channels, enabling cross-instance rendezvous.
type MessageStatusSync interface {
	// AddMessage registers that a message with the given wamID has been sent
	// and blocks until a corresponding status arrives (or times out).
	AddMessage(wamID string, timeout time.Duration) error

	// MessageSaved signals that the message has been persisted with the given
	// database ID. Must be called after AddMessage returns successfully.
	MessageSaved(wamID string, messageID uuid.UUID, timeout time.Duration) error

	// RollbackMessage signals that the message could not be persisted.
	// The waiting AddStatus call will receive an empty ID and return an error.
	RollbackMessage(wamID string, timeout time.Duration) error

	// AddStatus registers that a status update for wamID has arrived and blocks
	// until the message is saved (or rolled back, or times out).
	// Returns the database message ID on success.
	AddStatus(wamID string, status *message_model.SendingStatus, timeout time.Duration) (uuid.UUID, error)
}
