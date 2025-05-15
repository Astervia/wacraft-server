package message_service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	message_model "github.com/Rfluid/whatsapp-cloud-api/src/message/model"
	"github.com/google/uuid"
)

type MessageStatusSynchronizer struct {
	channels map[string]*chan string // Channels used to wait for message and status. The key is the message ID.
	mu       *sync.Mutex
}

// Used to signal that a message is waiting for a status to be saved to db.
// You must call MessageSaved after this function to avoid undefined behaviors.
func (m *MessageStatusSynchronizer) AddMessage(wamId string, timeout time.Duration) error {
	m.mu.Lock()
	ch, ok := m.channels[wamId]
	if !ok {
		chToP := make(chan string)
		ch = &chToP
		m.channels[wamId] = ch
	}
	m.mu.Unlock()

	select {
	case <-*ch: // Wait for status to be added
		return nil
	case <-time.After(timeout):
		// Remove the channel from the map to prevent leaks
		m.mu.Lock()
		delete(m.channels, wamId)
		m.mu.Unlock()
		return fmt.Errorf(
			"timeout waiting for whatsapp message status update. Waited %s for WhatsApp to update the message status and did not receive any updates",
			timeout.String())
	}
}

func (m *MessageStatusSynchronizer) RollbackMessage(wamId string, timeout time.Duration) error {
	defer func() {
		m.mu.Lock()
		delete(m.channels, wamId)
		m.mu.Unlock()
	}()
	m.mu.Lock()
	ch := m.channels[wamId]
	m.mu.Unlock()
	select {
	case *ch <- "": // Signal that message was rolledback and propagate empty string
		return nil
	case <-time.After(timeout):
		return fmt.Errorf(
			"timeout waiting to signal message rolledback. Signaling that the message was rolledback took %s and no status update was found for this message",
			timeout.String())
	}
}

// Signals that a message was saved to db and propagates its id.
func (m *MessageStatusSynchronizer) MessageSaved(wamId string, messageId uuid.UUID, timeout time.Duration) error {
	defer func() {
		m.mu.Lock()
		delete(m.channels, wamId)
		m.mu.Unlock()
	}()
	m.mu.Lock()
	ch := m.channels[wamId]
	m.mu.Unlock()
	select {
	case *ch <- messageId.String(): // Signal that message was saved and propagate its id
		return nil
	case <-time.After(timeout):
		return fmt.Errorf(
			"timeout waiting to signal message saved. Signaling that the message was saved took %s and no status update was found for this message",
			timeout.String())
	}
}

// Used to signal that a status is waiting for a message to be saved to db.
func (m *MessageStatusSynchronizer) AddStatus(wamId string, status *message_model.SendingStatus, timeout time.Duration) (uuid.UUID, error) {
	if status == nil {
		return uuid.Nil, errors.New("status is nil")
	}

	m.mu.Lock()
	ch, ok := m.channels[wamId]
	if !ok {
		chToP := make(chan string)
		ch = &chToP
		m.channels[wamId] = ch
	}
	m.mu.Unlock()

	// Signal that status was added
	select {
	case *ch <- "":
	case <-time.After(timeout):
		// Remove the channel from the map to prevent leaks
		m.mu.Lock()
		delete(m.channels, wamId)
		m.mu.Unlock()
		return uuid.Nil,
			fmt.Errorf(
				"timeout waiting to signal status added. Signaling that the status was added took %s and no message was found waiting for this status",
				timeout.String())
	}

	// Wait for message to be added and get the message ID
	var messageId string
	select {
	case messageId = <-*ch:
	case <-time.After(timeout):
		m.mu.Lock()
		delete(m.channels, wamId)
		m.mu.Unlock()
		return uuid.Nil,
			fmt.Errorf(
				"timeout waiting to message added. Waiting for message to be added took %s and no message was added for this status",
				timeout.String())
	}

	if messageId == "" {
		return uuid.Nil, errors.New("message rolled back")
	}
	messageIdAsUuid, err := uuid.Parse(messageId)
	if err != nil {
		return uuid.Nil, err
	}

	return messageIdAsUuid, nil
}

func CreateMessageStatusSynchronizer() *MessageStatusSynchronizer {
	return &MessageStatusSynchronizer{
		channels: make(map[string]*chan string),
		mu:       &sync.Mutex{},
	}
}

var StatusSynchronizer = CreateMessageStatusSynchronizer()
