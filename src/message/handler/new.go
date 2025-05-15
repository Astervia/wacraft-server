package message_handler

import (
	"sync"

	message_entity "github.com/Astervia/wacraft-core/src/message/entity"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	websocket_model "github.com/Astervia/wacraft-core/src/websocket/model"
	"github.com/gofiber/contrib/websocket"
)

var (
	newMessageClientPool = websocket_model.CreateClientPool()
	NewMessageChannel    = websocket_model.CreateChannel[websocket_model.ClientId, message_entity.Message, string]()
)

// NewMessageSubscription upgrades the connection to WebSocket and streams new WhatsApp messages.
//	@Summary		Watches for new messages
//	@Description	Upgrades the connection to WebSocket. Streams messages sent or received in real-time.
//	@Tags			Message Websocket
//	@Accept			json
//	@Produce		json
//	@Success		101	{string}	string							"WebSocket connection established"
//	@Failure		400	{object}	common_model.DescriptiveError	"Invalid connection request"
//	@Failure		500	{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/websocket/message/new [get]
func NewMessageSubscription(ctx *websocket.Conn) {
	defer ctx.Close()

	// Registering user
	user := ctx.Locals("user").(*user_entity.User) // This must be paired with the UserMiddleware. Otherwise will panic.
	clientId := newMessageClientPool.CreateId(user.Id)
	client := websocket_model.Client[websocket_model.ClientId]{
		Connection: ctx,
		Data:       *clientId,
	}
	NewMessageChannel.AppendClient(client, clientId.String())

	// Configuring disconnection
	defer func() {
		var deleteWg sync.WaitGroup

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			newMessageClientPool.DeleteId(*clientId)
		}()

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			NewMessageChannel.RemoveClient(client.Data.String())
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
