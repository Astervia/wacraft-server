package user_handler

import (
	"errors"

	auth_model "github.com/Astervia/wacraft-core/src/auth/model"
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	auth_service "github.com/Astervia/wacraft-server/src/auth/service"
	"github.com/gofiber/fiber/v2"
)

// OAuthTokenHandler handles the OAuth 2.0 token exchange.
//
//	@Summary		Issue access and refresh tokens
//	@Description	Issues access and refresh tokens based on the provided grant type (password or refresh_token).
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		auth_model.TokenRequest			true	"OAuth token request"
//	@Success		200			{object}	auth_model.TokenResponse		"Token issued successfully"
//	@Failure		400			{object}	common_model.DescriptiveError	"Bad request or missing fields"
//	@Failure		401			{object}	common_model.DescriptiveError	"Unauthorized or invalid credentials"
//	@Failure		500			{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/user/oauth/token [post]
//	@Security		ApiKeyAuth
func OAuthTokenHandler(c *fiber.Ctx) error {
	var req auth_model.TokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(common_model.NewParseJsonError(err).Send())
	}

	switch req.GrantType {
	case "password":
		if req.Username == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(
				common_model.NewParseJsonError(errors.New("missing username or password")).Send(),
			)
		}
		return handlePasswordGrant(c, req.Username, req.Password)

	case "refresh_token":
		if req.RefreshToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(
				common_model.NewParseJsonError(errors.New("missing refresh token")).Send(),
			)
		}
		return handleRefreshTokenGrant(c, req.RefreshToken)

	default:
		return c.Status(fiber.StatusBadRequest).SendString("unsupported grant type")
	}
}

func handlePasswordGrant(c *fiber.Ctx, email, password string) error {
	token, err := auth_service.Login(email, password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("unable to login", err, "auth_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(token)
}

func handleRefreshTokenGrant(c *fiber.Ctx, refreshToken string) error {
	token, err := auth_service.RefreshToken(refreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("unable to refresh token", err, "auth_service").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(token)
}
