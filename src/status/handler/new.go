package status_handler

import (
	"sync"

	status_entity "github.com/Astervia/wacraft-core/src/status/entity"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	websocket_model "github.com/Astervia/wacraft-core/src/websocket/model"
	"github.com/gofiber/contrib/websocket"
)

var (
	newStatusClientPool = websocket_model.CreateClientPool()
	NewStatusChannel    = websocket_model.CreateChannel[websocket_model.ClientId, status_entity.Status, string]()
)

// NewStatusSubscription establishes a WebSocket connection to receive real-time status updates.
//
//	@Summary		Subscribe to status updates
//	@Description	Upgrades the HTTP connection to a WebSocket stream to receive real-time message status updates.
//	@Tags			Status Websocket
//	@Accept			json
//	@Produce		json
//	@Success		101	{string}	string							"WebSocket connection established"
//	@Failure		400	{object}	common_model.DescriptiveError	"Invalid WebSocket handshake"
//	@Failure		401	{object}	common_model.DescriptiveError	"Unauthorized"
//	@Failure		500	{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/websocket/status/new [get]
func NewStatusSubscription(ctx *websocket.Conn) {
	defer ctx.Close()

	user := ctx.Locals("user").(*user_entity.User)
	clientId := newStatusClientPool.CreateId(user.Id)
	client := websocket_model.Client[websocket_model.ClientId]{
		Connection: ctx,
		Data:       *clientId,
	}
	NewStatusChannel.AppendClient(client, clientId.String())

	defer func() {
		var deleteWg sync.WaitGroup

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			newStatusClientPool.DeleteId(*clientId)
		}()

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			NewStatusChannel.RemoveClient(client.Data.String())
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
