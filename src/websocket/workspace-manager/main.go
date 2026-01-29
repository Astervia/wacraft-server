package websocket_workspace_manager

import (
	"sync"

	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	websocket_model "github.com/Astervia/wacraft-core/src/websocket/model"
	"github.com/google/uuid"
)

// WorkspaceChannelManager manages separate WebSocket channels per workspace using the mutex-swapper pattern.
// This ensures workspace-scoped broadcasting for messages and statuses.
type WorkspaceChannelManager[T any] struct {
	channels       map[uuid.UUID]*websocket_model.Channel[websocket_model.ClientID, T, string]
	channelSwapper *synch_service.MutexSwapper[uuid.UUID]
	globalMutex    sync.RWMutex
}

// GetOrCreateChannel retrieves or creates a channel for a specific workspace.
// Uses mutex-swapper pattern to safely manage concurrent access.
func (m *WorkspaceChannelManager[T]) GetOrCreateChannel(workspaceID uuid.UUID) *websocket_model.Channel[websocket_model.ClientID, T, string] {
	m.channelSwapper.Lock(workspaceID)
	defer m.channelSwapper.Unlock(workspaceID)

	m.globalMutex.Lock()
	defer m.globalMutex.Unlock()

	channel, exists := m.channels[workspaceID]
	if !exists {
		channel = websocket_model.CreateChannel[websocket_model.ClientID, T, string]()
		m.channels[workspaceID] = channel
	}

	return channel
}

// GetChannel retrieves a channel for a specific workspace if it exists.
// Returns nil if the workspace channel doesn't exist.
func (m *WorkspaceChannelManager[T]) GetChannel(workspaceID uuid.UUID) *websocket_model.Channel[websocket_model.ClientID, T, string] {
	m.globalMutex.RLock()
	defer m.globalMutex.RUnlock()

	return m.channels[workspaceID]
}

// BroadcastToWorkspace broadcasts data to all clients in a specific workspace.
func (m *WorkspaceChannelManager[T]) BroadcastToWorkspace(workspaceID uuid.UUID, data T) {
	channel := m.GetChannel(workspaceID)
	if channel != nil {
		channel.BroadcastJsonMultithread(data)
	}
}

// AppendClient adds a client to the workspace channel.
func (m *WorkspaceChannelManager[T]) AppendClient(workspaceID uuid.UUID, client websocket_model.Client[websocket_model.ClientID], key string) {
	channel := m.GetOrCreateChannel(workspaceID)
	channel.AppendClient(client, key)
}

// RemoveClient removes a client from the workspace channel.
func (m *WorkspaceChannelManager[T]) RemoveClient(workspaceID uuid.UUID, key string) {
	channel := m.GetChannel(workspaceID)
	if channel != nil {
		channel.RemoveClient(key)
	}
}

// CreateWorkspaceChannelManager creates a new workspace channel manager.
func CreateWorkspaceChannelManager[T any]() *WorkspaceChannelManager[T] {
	return &WorkspaceChannelManager[T]{
		channels:       make(map[uuid.UUID]*websocket_model.Channel[websocket_model.ClientID, T, string]),
		channelSwapper: synch_service.CreateMutexSwapper[uuid.UUID](),
	}
}
