package message_handler

import (
	"sync"

	_ "github.com/Astervia/wacraft-core/src/common/model"
	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	websocket_model "github.com/Astervia/wacraft-core/src/websocket/model"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	websocket_workspace_manager "github.com/Astervia/wacraft-server/src/websocket/workspace-manager"
	"github.com/gofiber/contrib/websocket"
)

// newMessageClientPool maintains all WebSocket clients connected for new messages
var (
	newMessageClientPool       = websocket_model.CreateClientPool()
	NewMessageWorkspaceManager = websocket_workspace_manager.CreateWorkspaceChannelManager[message_entity.Message]()
)

// NewMessageSubscription upgrades the connection to WebSocket and streams new WhatsApp messages.
//
//	@Summary		Subscribe to new messages
//	@Description	Establishes a WebSocket connection and streams incoming and outgoing WhatsApp messages in real-time for a specific workspace.
//	@Tags			Message Websocket
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	query		string							false	"Workspace ID (alternative to header)"
//	@Success		101				{string}	string							"WebSocket connection established"
//	@Failure		400				{object}	common_model.DescriptiveError	"Invalid connection request"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/websocket/message/new [get]
func NewMessageSubscription(ctx *websocket.Conn) {
	defer ctx.Close()

	// Registering user and workspace
	user := ctx.Locals("user").(*user_entity.User)                     // This must be paired with the UserMiddleware. Otherwise will panic.
	workspace := ctx.Locals("workspace").(*workspace_entity.Workspace) // This must be paired with the WebSocketWorkspaceMiddleware. Otherwise will panic.

	clientID := newMessageClientPool.CreateID(user.ID)
	client := websocket_model.Client[websocket_model.ClientID]{
		Connection: ctx,
		Data:       *clientID,
	}
	NewMessageWorkspaceManager.AppendClient(workspace.ID, client, clientID.String())

	// Configuring disconnection
	defer func() {
		var deleteWg sync.WaitGroup

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			newMessageClientPool.DeleteID(*clientID)
		}()

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			NewMessageWorkspaceManager.RemoveClient(workspace.ID, client.Data.String())
		}()

		deleteWg.Wait()
	}()

	for {
		// Read message from WebSocket
		msgType, data, err := ctx.ReadMessage()
		if err != nil {
			break // connection closed or other error
		}

		// Only handle text frames; ignore others
		if msgType == websocket.TextMessage && string(data) == string(websocket_model.Ping) {
			// Send “pong” back to the same client
			if writeErr := ctx.WriteMessage(websocket.TextMessage, []byte(websocket_model.Pong)); writeErr != nil {
				break // stop if the write fails
			}
		}
	}
}
