package campaign_handler

import (
	"errors"
	"sync"

	campaign_entity "github.com/Astervia/wacraft-core/src/campaign/entity"
	campaign_model "github.com/Astervia/wacraft-core/src/campaign/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	websocket_model "github.com/Astervia/wacraft-core/src/websocket/model"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	campaign_service "github.com/Astervia/wacraft-server/src/campaign/service"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

var (
	sendCampaignClientPool = websocket_model.CreateClientPool()
	SendCampaignPool       = campaign_model.CreateChannelPool()
)

// SendWhatsAppCampaignSubscription upgrades the connection to WebSocket and streams WhatsApp campaign results.
//
//	@Summary		Send WhatsApp campaign via WebSocket
//	@Description	Upgrades the connection to WebSocket to start sending a campaign and receive real-time status and results. Use message types: "send", "cancel", "status", and "ping".
//	@Tags			Campaign Websocket
//	@Accept			json
//	@Produce		json
//	@Param			campaignID	path		string							true	"Campaign ID (UUID format)"
//	@Param			function	body		campaign_model.SendMessage		false	"Optional: customize message sending behavior (currently not used)"
//	@Success		101			{string}	string							"WebSocket connection established"
//	@Failure		400			{object}	common_model.DescriptiveError	"Invalid campaign ID or bad request"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/websocket/campaign/whatsapp/send/{campaignID} [get]
func SendWhatsAppCampaignSubscription(ctx *websocket.Conn) {
	defer ctx.Close()

	campaignID, err := uuid.Parse(ctx.Params("campaignID"))
	if err != nil {
		ctx.WriteJSON(common_model.NewApiError("unable to parse campaign id", err, "handler").Send())
		return
	}

	workspace := ctx.Locals("workspace").(*workspace_entity.Workspace)
	user := ctx.Locals("user").(*user_entity.User)

	// Validate campaign belongs to workspace
	var campaign campaign_entity.Campaign
	if err := database.DB.Where("id = ? AND workspace_id = ?", campaignID, workspace.ID).First(&campaign).Error; err != nil {
		ctx.WriteJSON(common_model.NewApiError("campaign not found or access denied", err, "handler").Send())
		return
	}

	clientID := sendCampaignClientPool.CreateID(user.ID)
	client := websocket_model.CreateClient(*clientID, ctx)
	campaignChannel := SendCampaignPool.AddUser(*client, clientID.String(), campaignID, nil)

	defer func() {
		var deleteWg sync.WaitGroup

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			sendCampaignClientPool.DeleteID(*clientID)
		}()

		deleteWg.Add(1)
		go func() {
			defer deleteWg.Done()
			SendCampaignPool.RemoveUser(clientID.String(), campaignID)
		}()

		deleteWg.Wait()
	}()

	for {
		messageType, message, err := ctx.ReadMessage()
		if err != nil {
			break
		}

		err = handleSendWhatsAppCampaignMessage(campaignID, messageType, message, *campaignChannel, *client)
		if err != nil {
			ctx.WriteJSON(common_model.NewApiError("unable to handle message", err, "handler").Send())
		}
	}
}

func watchWhatsAppCampaignResults(
	resultsCh <-chan campaign_model.CampaignResults,
	ctx *websocket.Conn,
) {
	for result := range resultsCh {
		ctx.WriteJSON(result)
	}
}

func handleSendWhatsAppCampaignMessage(
	campaignID uuid.UUID,
	messageType int,
	message []byte,
	campaignChannel campaign_model.CampaignChannel,
	client websocket_model.Client[websocket_model.ClientID],
) error {
	if messageType != websocket.TextMessage {
		return errors.New("only text messages are allowed")
	}

	switch string(message) {
	case string(websocket_model.Ping):
		return client.Connection.WriteMessage(websocket.TextMessage, []byte(websocket_model.Pong))

	case string(campaign_model.Send):
		if campaignChannel.Sending {
			return errors.New("currently sending campaign")
		}
		_, err := campaign_service.SendWhatsAppCampaign(
			campaignID,
			campaignChannel,
			func(data *campaign_model.CampaignResults) {
				campaignChannel.BroadcastJsonMultithread(*data)
			},
		)
		return err

	case string(campaign_model.Cancel):
		err := campaignChannel.Cancel()
		if err != nil {
			return err
		}
		campaignChannel.BroadcastMessageMultithread(websocket.TextMessage, []byte(campaign_model.NotSending))
		return nil

	case string(campaign_model.Status):
		status := campaign_model.NotSending
		if campaignChannel.Sending {
			status = campaign_model.Sending
		}
		return client.Connection.WriteMessage(websocket.TextMessage, []byte(status))
	}

	return errors.New("unsupported message")
}
