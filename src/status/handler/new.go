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
//	@Summary		Watches for new statuses
//	@Description	WebSocket route that allows the user to watch for incoming message status updates.
//	@Tags			Status Websocket
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	status_entity.Status			"Status update received"
//	@Failure		400	{object}	common_model.DescriptiveError	"Invalid WebSocket handshake"
//	@Failure		401	{object}	common_model.DescriptiveError	"Unauthorized"
//	@Failure		500	{object}	common_model.DescriptiveError	"Server error"
//	@Security		ApiKeyAuth
//	@Router			/websocket/status/new [get]
func NewStatusSubscription(ctx *websocket.Conn) {
	defer ctx.Close()

	// Registering user
	user := ctx.Locals("user").(*user_entity.User) // This must be paired with the UserMiddleware. Otherwise will panic.
	clientId := newStatusClientPool.CreateId(user.Id)
	client := websocket_model.Client[websocket_model.ClientId]{
		Connection: ctx,
		Data:       *clientId,
	}
	NewStatusChannel.AppendClient(client, clientId.String())

	// Configuring disconnection
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
