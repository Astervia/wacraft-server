package media_handler

import (
	"errors"
	"strconv"

	cmn_model "github.com/Astervia/wacraft-core/src/common/model"
	common_service "github.com/Astervia/wacraft-core/src/common/service"
	messaging_product_entity "github.com/Astervia/wacraft-core/src/messaging-product/entity"
	messaging_product_model "github.com/Astervia/wacraft-core/src/messaging-product/model"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/integration/whatsapp"
	phone_config_service "github.com/Astervia/wacraft-server/src/phone-config/service"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	bootstrap_module "github.com/Rfluid/whatsapp-cloud-api/src/bootstrap"
	common_model "github.com/Rfluid/whatsapp-cloud-api/src/common"
	media_model "github.com/Rfluid/whatsapp-cloud-api/src/media"
	media_service "github.com/Rfluid/whatsapp-cloud-api/src/media"
	"github.com/gofiber/fiber/v2"
)

// getWorkspaceWhatsAppAPI retrieves the workspace-specific WhatsApp API client
// Falls back to global API if no phone config is configured for the workspace
func getWorkspaceWhatsAppAPI(workspace *workspace_entity.Workspace) *bootstrap_module.WhatsAppAPI {
	// Find messaging product for workspace
	mp := messaging_product_entity.MessagingProduct{
		Name:        messaging_product_model.WhatsApp,
		WorkspaceID: &workspace.ID,
	}

	if err := database.DB.Model(&mp).Where(&mp).First(&mp).Error; err != nil {
		// No messaging product found, use global API
		return &whatsapp.WabaApi
	}

	// If phone config is available, use workspace-specific API
	if mp.PhoneConfigID != nil {
		wabaApi, err := phone_config_service.GetWhatsAppAPIByPhoneConfigID(*mp.PhoneConfigID)
		if err == nil {
			return wabaApi
		}
	}

	// Fallback to global API
	return &whatsapp.WabaApi
}

// GetWhatsAppMediaURL retrieves a temporary download URL for a WhatsApp media item.
//
//	@Summary		Get WhatsApp media URL
//	@Description	Uses the WhatsApp API to retrieve a temporary media download URL. This URL expires in 5 minutes.
//	@Tags			Media
//	@Accept			json
//	@Produce		json
//	@Param			mediaID	path		string							true	"Media ID"
//	@Success		200		{object}	media_model.MediaInfo			"Media information with download URL"
//	@Failure		400		{object}	cmn_model.DescriptiveError	"Missing or invalid media ID"
//	@Failure		500		{object}	cmn_model.DescriptiveError	"Failed to retrieve media URL"
//	@Security		ApiKeyAuth
//	@Router			/media/whatsapp/{mediaID} [get]
func GetWhatsAppMediaURL(ctx *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(ctx)
	wabaApi := getWorkspaceWhatsAppAPI(workspace)

	mediaID := ctx.Params("mediaID")
	if mediaID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(
			cmn_model.NewApiError("media ID is required", errors.New("no media ID provided"), "handler").Send(),
		)
	}

	mediaInfo, err := media_service.RetrieveURL(*wabaApi, mediaID, media_model.RetrieveInfo{})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(
			cmn_model.NewApiError("failed to retrieve media URL", err, "handler").Send(),
		)
	}

	return ctx.Status(fiber.StatusOK).JSON(mediaInfo)
}

// DownloadWhatsAppMedia downloads a media file directly from WhatsApp using its media ID.
//
//	@Summary		Download WhatsApp media
//	@Description	Downloads media using a temporary URL retrieved via the WhatsApp API.
//	@Tags			Media
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			mediaID	path		string							true	"Media ID"
//	@Success		200		{file}		binary							"Downloaded media file"
//	@Failure		400		{object}	cmn_model.DescriptiveError	"Missing or invalid media ID"
//	@Failure		500		{object}	cmn_model.DescriptiveError	"Failed to download media"
//	@Security		ApiKeyAuth
//	@Router			/media/whatsapp/download/{mediaID} [get]
func DownloadWhatsAppMedia(ctx *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(ctx)
	wabaApi := getWorkspaceWhatsAppAPI(workspace)

	mediaID := ctx.Params("mediaID")
	if mediaID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(
			cmn_model.NewApiError("media ID is required", errors.New("no media ID provided"), "handler").Send(),
		)
	}

	mediaInfo, err := media_service.RetrieveURL(*wabaApi, mediaID, media_model.RetrieveInfo{})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(
			cmn_model.NewApiError("failed to retrieve media URL", err, "service").Send(),
		)
	}

	mediaBytes, err := media_service.Download(*wabaApi, mediaInfo.URL)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(
			cmn_model.NewApiError("failed to download media", err, "service").Send(),
		)
	}

	ctx.Set("Content-Type", mediaInfo.MimeType)
	ctx.Set("Content-Disposition", "attachment; filename="+mediaID+"."+common_service.GetExtensionFromMimeType(mediaInfo.MimeType))
	ctx.Set("Content-Length", strconv.FormatInt(mediaInfo.FileSize, 10))

	return ctx.Send(mediaBytes)
}

// DownloadFromMediaInfo downloads media based on information in the request body.
//
//	@Summary		Download media from MediaInfo
//	@Description	Receives MediaInfo JSON, validates it, downloads the media from the provided URL, and streams it.
//	@Tags			Media
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			mediaInfo	body		media_model.MediaInfo			true	"Media Info with URL and metadata"
//	@Success		200			{file}		binary							"Downloaded media file"
//	@Failure		400		{object}	cmn_model.DescriptiveError	"Invalid MediaInfo"
//	@Failure		500		{object}	cmn_model.DescriptiveError	"Failed to download media"
//	@Security		ApiKeyAuth
//	@Router			/media/whatsapp/media-info/download [post]
func DownloadFromMediaInfo(ctx *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(ctx)
	wabaApi := getWorkspaceWhatsAppAPI(workspace)

	var mediaInfo media_model.MediaInfo
	if err := ctx.BodyParser(&mediaInfo); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(
			cmn_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&mediaInfo); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(
			cmn_model.NewValidationError(err).Send(),
		)
	}

	mediaBytes, err := media_service.Download(*wabaApi, mediaInfo.URL)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(
			cmn_model.NewApiError("failed to download media", err, "handler").Send(),
		)
	}

	ctx.Set("Content-Type", mediaInfo.MimeType)
	ctx.Set("Content-Disposition", "attachment; filename="+mediaInfo.ID.ID+"."+common_service.GetExtensionFromMimeType(mediaInfo.MimeType))
	ctx.Set("Content-Length", strconv.FormatInt(mediaInfo.FileSize, 10))

	return ctx.Send(mediaBytes)
}

// UploadWhatsAppMedia uploads a media file to WhatsApp.
//
//	@Summary		Upload media file
//	@Description	Uploads a media file to WhatsApp. Files remain available for up to 30 days unless deleted earlier.
//	@Tags			Media
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file							true	"Media file"
//	@Param			type	formData	string							true	"MIME type of the media file"
//	@Success		200		{object}	common_model.ID					"Media ID returned from WhatsApp"
//	@Failure		400		{object}	cmn_model.DescriptiveError	"Missing file or MIME type"
//	@Failure		415		{object}	cmn_model.DescriptiveError	"Unsupported media type"
//	@Failure		500		{object}	cmn_model.DescriptiveError	"Failed to upload media"
//	@Security		ApiKeyAuth
//	@Router			/media/whatsapp/upload [post]
func UploadWhatsAppMedia(ctx *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(ctx)
	wabaApi := getWorkspaceWhatsAppAPI(workspace)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(
			cmn_model.NewApiError("file is required", err, "handler").Send(),
		)
	}

	mimeType := ctx.FormValue("type")
	if mimeType == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(
			cmn_model.NewApiError("MIME type is required", errors.New("no type provided"), "handler").Send(),
		)
	}

	supportedMimeType, err := common_model.ParseMimeType(mimeType)
	if err != nil {
		return ctx.Status(fiber.StatusUnsupportedMediaType).JSON(
			cmn_model.NewApiError("unsupported MIME type", err, "handler").Send(),
		)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(
			cmn_model.NewApiError("unable to open file", err, "handler").Send(),
		)
	}
	defer file.Close()

	uploadData := media_model.Upload{
		FileName: fileHeader.Filename,
		FileData: file,
		Type:     supportedMimeType,
	}
	uploadData.SetDefault()

	mediaID, err := media_service.Upload(*wabaApi, uploadData)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(
			cmn_model.NewApiError("failed to upload media", err, "handler").Send(),
		)
	}

	return ctx.Status(fiber.StatusOK).JSON(mediaID)
}
