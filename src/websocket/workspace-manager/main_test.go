package websocket_workspace_manager

import (
	"encoding/json"
	"testing"
	"time"

	synch_service "github.com/Astervia/wacraft-core/src/synch/service"
	websocket_model "github.com/Astervia/wacraft-core/src/websocket/model"
	"github.com/google/uuid"
)

type testMsg struct {
	Value string `json:"value"`
}

func makeTestClient() websocket_model.Client[websocket_model.ClientID] {
	return websocket_model.Client[websocket_model.ClientID]{
		Data: websocket_model.ClientID{UserID: uuid.New()},
	}
}

// TestWorkspaceManager_LocalBroadcast verifies that BroadcastToWorkspace
// publishes to the PubSub backend when one is configured.
func TestWorkspaceManager_LocalBroadcast(t *testing.T) {
	pubsub := synch_service.NewMemoryPubSub()
	workspaceID := uuid.New()

	mgr := CreateWorkspaceChannelManager[testMsg]()
	mgr.SetPubSub(pubsub, "workspace:test")

	// Subscribe directly to capture what BroadcastToWorkspace publishes.
	sub, err := pubsub.Subscribe(mgr.pubsubChannel(workspaceID))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	want := testMsg{Value: "hello-local"}
	mgr.BroadcastToWorkspace(workspaceID, want)

	select {
	case raw := <-sub.Channel():
		var got testMsg
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Value != want.Value {
			t.Fatalf("got %q, want %q", got.Value, want.Value)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast message")
	}
}

// TestWorkspaceManager_CrossInstanceBroadcast verifies that a message published
// by one manager instance is received by a subscriber on the shared PubSub
// backend, simulating delivery to another instance.
func TestWorkspaceManager_CrossInstanceBroadcast(t *testing.T) {
	pubsub := synch_service.NewMemoryPubSub()
	workspaceID := uuid.New()

	mgrA := CreateWorkspaceChannelManager[testMsg]()
	mgrA.SetPubSub(pubsub, "workspace:test")

	mgrB := CreateWorkspaceChannelManager[testMsg]()
	mgrB.SetPubSub(pubsub, "workspace:test")

	// Subscribe on the shared PubSub to intercept what mgrB publishes —
	// this simulates mgrA's internal subscriber goroutine.
	sub, err := pubsub.Subscribe(mgrA.pubsubChannel(workspaceID))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	want := testMsg{Value: "cross-instance"}
	mgrB.BroadcastToWorkspace(workspaceID, want)

	select {
	case raw := <-sub.Channel():
		var got testMsg
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Value != want.Value {
			t.Fatalf("got %q, want %q", got.Value, want.Value)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for cross-instance broadcast")
	}
}

// TestWorkspaceManager_SubscribeOnConnect verifies that a PubSub subscription
// is created when the first client joins a workspace and that a second client
// joining the same workspace reuses the existing subscription.
func TestWorkspaceManager_SubscribeOnConnect(t *testing.T) {
	pubsub := synch_service.NewMemoryPubSub()
	workspaceID := uuid.New()

	mgr := CreateWorkspaceChannelManager[testMsg]()
	mgr.SetPubSub(pubsub, "workspace:test")

	mgr.AppendClient(workspaceID, makeTestClient(), "client-1")

	mgr.globalMutex.RLock()
	_, hasSub := mgr.subscriptions[workspaceID]
	mgr.globalMutex.RUnlock()
	if !hasSub {
		t.Fatal("expected subscription after first client connects")
	}

	mgr.AppendClient(workspaceID, makeTestClient(), "client-2")

	mgr.globalMutex.RLock()
	subCount := len(mgr.subscriptions)
	mgr.globalMutex.RUnlock()
	if subCount != 1 {
		t.Fatalf("expected 1 subscription for 2 clients on same workspace, got %d", subCount)
	}

	// Clean up to stop the subscriber goroutine.
	mgr.RemoveClient(workspaceID, "client-1")
	mgr.RemoveClient(workspaceID, "client-2")
}

// TestWorkspaceManager_UnsubscribeOnLastDisconnect verifies that the PubSub
// subscription is kept alive while clients remain and is cancelled only when
// the last client disconnects.
func TestWorkspaceManager_UnsubscribeOnLastDisconnect(t *testing.T) {
	pubsub := synch_service.NewMemoryPubSub()
	workspaceID := uuid.New()

	mgr := CreateWorkspaceChannelManager[testMsg]()
	mgr.SetPubSub(pubsub, "workspace:test")

	mgr.AppendClient(workspaceID, makeTestClient(), "client-1")
	mgr.AppendClient(workspaceID, makeTestClient(), "client-2")

	// Remove one client — subscription must still be active.
	mgr.RemoveClient(workspaceID, "client-1")
	mgr.globalMutex.RLock()
	_, hasSub := mgr.subscriptions[workspaceID]
	mgr.globalMutex.RUnlock()
	if !hasSub {
		t.Fatal("subscription should still exist with one client remaining")
	}

	// Remove last client — subscription and channel must be cleaned up.
	mgr.RemoveClient(workspaceID, "client-2")
	mgr.globalMutex.RLock()
	_, hasSub = mgr.subscriptions[workspaceID]
	_, hasCh := mgr.channels[workspaceID]
	mgr.globalMutex.RUnlock()
	if hasSub {
		t.Fatal("subscription should have been removed after last client disconnects")
	}
	if hasCh {
		t.Fatal("channel should have been removed after last client disconnects")
	}
}

// TestWorkspaceManager_IsolatedWorkspaces verifies that a broadcast to workspace
// B is not received by a subscriber registered for workspace A.
func TestWorkspaceManager_IsolatedWorkspaces(t *testing.T) {
	pubsub := synch_service.NewMemoryPubSub()
	workspaceA := uuid.New()
	workspaceB := uuid.New()

	mgr := CreateWorkspaceChannelManager[testMsg]()
	mgr.SetPubSub(pubsub, "workspace:test")

	subA, err := pubsub.Subscribe(mgr.pubsubChannel(workspaceA))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer subA.Unsubscribe()

	mgr.BroadcastToWorkspace(workspaceB, testMsg{Value: "for-B-only"})

	select {
	case msg := <-subA.Channel():
		t.Fatalf("workspace A should not have received a message for workspace B: %s", msg)
	case <-time.After(100 * time.Millisecond):
		// expected — no cross-workspace leakage
	}
}
