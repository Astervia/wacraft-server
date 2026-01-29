package status_handler

import (
	"sync"

	_ "github.com/Astervia/wacraft-core/src/common/model"
	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	websocket_model "github.com/Astervia/wacraft-core/src/websocket/model"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	websocket_workspace_manager "github.com/Astervia/wacraft-server/src/websocket/workspace-manager"
	"github.com/gofiber/contrib/websocket"
)

var (
	newStatusClientPool       = websocket_model.CreateClientPool()
	NewStatusWorkspaceManager = websocket_workspace_manager.CreateWorkspaceChannelManager[status_entity.Status]()
)

// NewStatusSubscription establishes a WebSocket connection to receive real-time status updates.
//
//	@Summary		Subscribe to status updates
//	@Description	Upgrades the HTTP connection to a WebSocket stream to receive real-time message status updates for a specific workspace.
//	@Tags			Status Websocket
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	query		string							false	"Workspace ID (alternative to header)"
//	@Success		101				{string}	string							"WebSocket connection established"
//	@Failure		400				{object}	common_model.DescriptiveError	"Invalid WebSocket handshake"
//	@Failure		401				{object}	common_model.DescriptiveError	"Unauthorized"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/websocket/status/new [get]
func NewStatusSubscription(ctx *websocket.Conn) {
	defer ctx.Close()

	user := ctx.Locals("user").(*user_entity.User)
	workspace := ctx.Locals("workspace").(*workspace_entity.Workspace)

	clientID := newStatusClientPool.CreateID(user.ID)
	client := websocket_model.Client[websocket_model.ClientID]{
		Connection: ctx,
		Data:       *clientID,
	}
	NewStatusWorkspaceManager.AppendClient(workspace.ID, client, clientID.String())

	defer func() {
		var deleteWg sync.WaitGroup

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			newStatusClientPool.DeleteID(*clientID)
		}()

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			NewStatusWorkspaceManager.RemoveClient(workspace.ID, client.Data.String())
		}()

		deleteWg.Wait()
	}()

	for {
		msgType, data, err := ctx.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.TextMessage && string(data) == string(websocket_model.Ping) {
			if writeErr := ctx.WriteMessage(websocket.TextMessage, []byte(websocket_model.Pong)); writeErr != nil {
				break
			}
		}
	}
}
