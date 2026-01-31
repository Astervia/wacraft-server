package auth_handler

import (
	"strings"
	"time"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	crypto_service "github.com/Astervia/wacraft-core/src/crypto/service"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/Astervia/wacraft-server/src/database"
	email_service "github.com/Astervia/wacraft-server/src/email/service"
	"github.com/Astervia/wacraft-server/src/validators"
	"github.com/gofiber/fiber/v2"
)

// ForgotPassword initiates a password reset request.
//
//	@Summary		Request password reset
//	@Description	Sends a password reset email to the user if the email exists.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		user_model.ForgotPasswordRequest	true	"Email address"
//	@Success		200		{object}	user_model.ForgotPasswordResponse	"Password reset email sent"
//	@Failure		400		{object}	common_model.DescriptiveError		"Invalid request"
//	@Failure		500		{object}	common_model.DescriptiveError		"Internal server error"
//	@Router			/auth/forgot-password [post]
func ForgotPassword(c *fiber.Ctx) error {
	var req user_model.ForgotPasswordRequest
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

	// Always return success to prevent email enumeration
	successResponse := user_model.ForgotPasswordResponse{
		Message: "If your email is registered, you will receive a password reset link",
	}

	// Find user
	var user user_entity.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusOK).JSON(successResponse)
	}

	// Generate reset token
	token, err := crypto_service.GeneratePasswordResetToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to generate reset token", err, "crypto").Send(),
		)
	}

	// Invalidate any existing tokens for this user
	database.DB.Where("user_id = ? AND used_at IS NULL", user.ID).Delete(&user_entity.PasswordResetToken{})

	// Create new reset token
	resetToken := user_entity.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := database.DB.Create(&resetToken).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to create reset token", err, "database").Send(),
		)
	}

	// Send reset email (async)
	go func() {
		if err := email_service.DefaultEmailService.SendPasswordReset(user.Email, user.Name, token); err != nil {
			// Log error but don't fail
			_ = err
		}
	}()

	return c.Status(fiber.StatusOK).JSON(successResponse)
}

// ResetPassword resets the user's password using a token.
//
//	@Summary		Reset password
//	@Description	Resets the user's password using the token from the reset email.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		user_model.ResetPasswordRequest	true	"Reset token and new password"
//	@Success		200		{object}	user_model.ResetPasswordResponse	"Password reset successful"
//	@Failure		400		{object}	common_model.DescriptiveError		"Invalid or expired token"
//	@Failure		500		{object}	common_model.DescriptiveError		"Internal server error"
//	@Router			/auth/reset-password [post]
func ResetPassword(c *fiber.Ctx) error {
	var req user_model.ResetPasswordRequest
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

	// Find reset token
	var resetToken user_entity.PasswordResetToken
	if err := database.DB.Where("token = ?", req.Token).Preload("User").First(&resetToken).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid reset token", err, "auth").Send(),
		)
	}

	// Validate token
	if !resetToken.IsValid() {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Reset token is invalid or expired", nil, "auth").Send(),
		)
	}

	// Hash new password
	hashedPassword, err := crypto_service.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to hash password", err, "crypto").Send(),
		)
	}

	// Update password and mark token as used in transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to start transaction", tx.Error, "database").Send(),
		)
	}

	// Update user password
	if err := tx.Model(&user_entity.User{}).Where("id = ?", resetToken.UserID).Update("password", hashedPassword).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to update password", err, "database").Send(),
		)
	}

	// Mark token as used
	now := time.Now()
	resetToken.UsedAt = &now
	if err := tx.Save(&resetToken).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to update reset token", err, "database").Send(),
		)
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to commit transaction", err, "database").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(user_model.ResetPasswordResponse{
		Message: "Password reset successful",
	})
}
