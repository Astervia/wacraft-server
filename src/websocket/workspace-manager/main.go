package websocket_workspace_manager

import (
	"encoding/json"
	"sync"

	synch_contract "github.com/Astervia/wacraft-core/src/synch/contract"
	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	websocket_model "github.com/Astervia/wacraft-core/src/websocket/model"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
)

// WorkspaceChannelManager manages separate WebSocket channels per workspace.
// When a PubSub backend is configured (via SetPubSub), every broadcast is
// published to the distributed channel so all instances deliver it to their
// local WebSocket clients — enabling cross-instance real-time propagation.
type WorkspaceChannelManager[T any] struct {
	channels       map[uuid.UUID]*websocket_model.Channel[websocket_model.ClientID, T, string]
	subscriptions  map[uuid.UUID]synch_contract.Subscription
	channelSwapper *synch_service.MutexSwapper[uuid.UUID]
	globalMutex    sync.RWMutex

	pubsub        synch_contract.PubSub
	channelPrefix string
}

// SetPubSub wires a distributed pub/sub backend into the manager.
// Call this during application initialisation (before any clients connect).
func (m *WorkspaceChannelManager[T]) SetPubSub(pubsub synch_contract.PubSub, channelPrefix string) {
	m.globalMutex.Lock()
	defer m.globalMutex.Unlock()
	m.pubsub = pubsub
	m.channelPrefix = channelPrefix
}

func (m *WorkspaceChannelManager[T]) pubsubChannel(workspaceID uuid.UUID) string {
	return m.channelPrefix + ":" + workspaceID.String()
}

// GetChannel retrieves a channel for a specific workspace if it exists.
func (m *WorkspaceChannelManager[T]) GetChannel(workspaceID uuid.UUID) *websocket_model.Channel[websocket_model.ClientID, T, string] {
	m.globalMutex.RLock()
	defer m.globalMutex.RUnlock()
	return m.channels[workspaceID]
}

// BroadcastToWorkspace broadcasts data to all WebSocket clients in the workspace.
// When PubSub is configured the data is published to the distributed channel so
// every instance broadcasts to its own local clients (including this one via the
// subscriber goroutine). In memory-only mode a direct local broadcast is performed.
func (m *WorkspaceChannelManager[T]) BroadcastToWorkspace(workspaceID uuid.UUID, data T) {
	// ⚡ Bolt Optimization: Combine state and map lookups into a single mutex critical
	// section to reduce lock contention and eliminate redundant RWMutex operations.
	m.globalMutex.RLock()
	pubsub := m.pubsub
	channel := m.channels[workspaceID]
	m.globalMutex.RUnlock()

	if pubsub != nil {
		b, err := json.Marshal(data)
		if err != nil {
			pterm.DefaultLogger.Error("workspace broadcast marshal error: " + err.Error())
			return
		}
		if err := pubsub.Publish(m.pubsubChannel(workspaceID), b); err != nil {
			pterm.DefaultLogger.Error("workspace broadcast publish error: " + err.Error())
		}
		return
	}

	// Memory-only mode: direct local broadcast.
	if channel != nil {
		channel.BroadcastJsonMultithread(data)
	}
}

// AppendClient adds a client to the workspace channel.
// When this is the first client for a workspace and PubSub is configured,
// a subscription is started to receive cross-instance broadcasts.
func (m *WorkspaceChannelManager[T]) AppendClient(workspaceID uuid.UUID, client websocket_model.Client[websocket_model.ClientID], key string) {
	m.channelSwapper.Lock(workspaceID)
	defer m.channelSwapper.Unlock(workspaceID)

	m.globalMutex.Lock()
	channel, exists := m.channels[workspaceID]
	if !exists {
		channel = websocket_model.CreateChannel[websocket_model.ClientID, T, string]()
		m.channels[workspaceID] = channel
	}
	pubsub := m.pubsub
	m.globalMutex.Unlock()

	channel.AppendClient(client, key)

	if !exists && pubsub != nil {
		m.subscribeWorkspace(workspaceID, channel)
	}
}

// subscribeWorkspace subscribes to the PubSub channel for a workspace and
// forwards every received message to local WebSocket clients.
func (m *WorkspaceChannelManager[T]) subscribeWorkspace(
	workspaceID uuid.UUID,
	channel *websocket_model.Channel[websocket_model.ClientID, T, string],
) {
	sub, err := m.pubsub.Subscribe(m.pubsubChannel(workspaceID))
	if err != nil {
		pterm.DefaultLogger.Error("workspace pubsub subscribe error: " + err.Error())
		return
	}

	m.globalMutex.Lock()
	m.subscriptions[workspaceID] = sub
	m.globalMutex.Unlock()

	go func() {
		for msg := range sub.Channel() {
			var data T
			if err := json.Unmarshal(msg, &data); err != nil {
				pterm.DefaultLogger.Error("workspace pubsub unmarshal error: " + err.Error())
				continue
			}
			channel.BroadcastJsonMultithread(data)
		}
	}()
}

// RemoveClient removes a client from the workspace channel.
// When the last client disconnects the PubSub subscription is cancelled.
func (m *WorkspaceChannelManager[T]) RemoveClient(workspaceID uuid.UUID, key string) {
	m.channelSwapper.Lock(workspaceID)
	defer m.channelSwapper.Unlock(workspaceID)

	m.globalMutex.RLock()
	channel, exists := m.channels[workspaceID]
	m.globalMutex.RUnlock()
	if !exists {
		return
	}

	channel.RemoveClient(key)

	channel.ClientsMutex.Lock()
	clientCount := len(channel.Clients)
	channel.ClientsMutex.Unlock()

	if clientCount == 0 {
		m.globalMutex.Lock()
		delete(m.channels, workspaceID)
		sub, hasSub := m.subscriptions[workspaceID]
		if hasSub {
			delete(m.subscriptions, workspaceID)
		}
		m.globalMutex.Unlock()

		if hasSub {
			if err := sub.Unsubscribe(); err != nil {
				pterm.DefaultLogger.Error("workspace pubsub unsubscribe error: " + err.Error())
			}
		}
	}
}

// CreateWorkspaceChannelManager creates a new workspace channel manager.
func CreateWorkspaceChannelManager[T any]() *WorkspaceChannelManager[T] {
	return &WorkspaceChannelManager[T]{
		channels:       make(map[uuid.UUID]*websocket_model.Channel[websocket_model.ClientID, T, string]),
		subscriptions:  make(map[uuid.UUID]synch_contract.Subscription),
		channelSwapper: synch_service.CreateMutexSwapper[uuid.UUID](),
	}
}
