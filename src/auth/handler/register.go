package auth_handler

import (
	"strings"
	"time"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	crypto_service "github.com/Astervia/wacraft-core/src/crypto/service"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/Astervia/wacraft-server/src/database"
	email_service "github.com/Astervia/wacraft-server/src/email/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
	"github.com/gosimple/slug"
)

// Register creates a new user account.
//
//	@Summary		Register new user
//	@Description	Creates a new user account and sends verification email if required.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		user_model.RegisterRequest	true	"Registration data"
//	@Success		201		{object}	user_model.RegisterResponse	"User created"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid request"
//	@Failure		403		{object}	common_model.DescriptiveError	"Registration disabled"
//	@Failure		409		{object}	common_model.DescriptiveError	"Email already exists"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/auth/register [post]
func Register(c *fiber.Ctx) error {
	if !env.AllowRegistration {
		return c.Status(fiber.StatusForbidden).JSON(
			common_model.NewApiError("Registration is disabled", nil, "auth").Send(),
		)
	}

	var req user_model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	// Normalize email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Check if email already exists
	var existingUser user_entity.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(
			common_model.NewApiError("Email already registered", nil, "auth").Send(),
		)
	}

	// Create user in transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to start transaction", tx.Error, "database").Send(),
		)
	}

	defaultRole := user_model.User
	user := user_entity.User{
		Name:          req.Name,
		Email:         req.Email,
		Password:      req.Password,
		Role:          &defaultRole,
		EmailVerified: !env.RequireEmailVerification,
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to create user", err, "database").Send(),
		)
	}

	// Create personal workspace for user
	workspaceSlug := slug.Make(req.Name + "-workspace")
	// Ensure unique slug
	workspaceSlug = workspaceSlug + "-" + user.ID.String()[:8]

	workspace := workspace_entity.Workspace{
		Name:      req.Name + "'s Workspace",
		Slug:      workspaceSlug,
		CreatedBy: user.ID,
	}

	if err := tx.Create(&workspace).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to create workspace", err, "database").Send(),
		)
	}

	// Add user as workspace member
	member := workspace_entity.WorkspaceMember{
		WorkspaceID: workspace.ID,
		UserID:      user.ID,
	}

	if err := tx.Create(&member).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to create workspace membership", err, "database").Send(),
		)
	}

	// Assign all admin policies to user
	for _, policy := range workspace_model.AllPolicies {
		policyRecord := workspace_entity.WorkspaceMemberPolicy{
			WorkspaceMemberID: member.ID,
			Policy:            policy,
		}
		if err := tx.Create(&policyRecord).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("Failed to assign policies", err, "database").Send(),
			)
		}
	}

	// If email verification required, create verification token and send email
	if env.RequireEmailVerification {
		token, err := crypto_service.GenerateVerificationToken()
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("Failed to generate verification token", err, "crypto").Send(),
			)
		}

		verification := user_entity.EmailVerification{
			UserID:    user.ID,
			Token:     token,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Verified:  false,
		}

		if err := tx.Create(&verification).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("Failed to create verification record", err, "database").Send(),
			)
		}

		// Send verification email (async, don't block on failure)
		origin := strings.TrimRight(c.Get("Origin"), "/")
		go func() {
			if err := email_service.DefaultEmailService.SendVerificationEmail(user.Email, user.Name, token, origin); err != nil {
				// Log error but don't fail registration
				_ = err
			}
		}()
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to commit transaction", err, "database").Send(),
		)
	}

	message := "Registration successful"
	if env.RequireEmailVerification {
		message = "Registration successful. Please check your email to verify your account."
	}

	return c.Status(fiber.StatusCreated).JSON(user_model.RegisterResponse{
		Message: message,
		UserID:  user.ID.String(),
	})
}
