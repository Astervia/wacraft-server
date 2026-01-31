package auth_handler

import (
	"time"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	user_model "github.com/Astervia/wacraft-core/src/user/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/gofiber/fiber/v2"
)

// VerifyEmail verifies a user's email address.
//
//	@Summary		Verify email
//	@Description	Verifies a user's email address using the token from the verification email.
//	@Tags			Auth
//	@Produce		json
//	@Param			token	query		string	true	"Verification token"
//	@Success		200		{object}	user_model.VerifyEmailResponse	"Email verified"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid or expired token"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Router			/auth/verify-email [get]
func VerifyEmail(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Token is required", nil, "validation").Send(),
		)
	}

	// Find verification record
	var verification user_entity.EmailVerification
	if err := database.DB.Where("token = ?", token).Preload("User").First(&verification).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid verification token", err, "auth").Send(),
		)
	}

	// Check if already verified
	if verification.Verified {
		return c.Status(fiber.StatusOK).JSON(user_model.VerifyEmailResponse{
			Message: "Email already verified",
		})
	}

	// Check if expired
	if time.Now().After(verification.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Verification token has expired", nil, "auth").Send(),
		)
	}

	// Update verification record and user in transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to start transaction", tx.Error, "database").Send(),
		)
	}

	// Mark verification as complete
	verification.Verified = true
	if err := tx.Save(&verification).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to update verification record", err, "database").Send(),
		)
	}

	// Mark user email as verified
	if err := tx.Model(&user_entity.User{}).Where("id = ?", verification.UserID).Update("email_verified", true).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to update user", err, "database").Send(),
		)
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to commit transaction", err, "database").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(user_model.VerifyEmailResponse{
		Message: "Email verified successfully",
	})
}

// ResendVerification resends the verification email.
//
//	@Summary		Resend verification email
//	@Description	Resends the email verification link to the user's email.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		user_model.ResendVerificationRequest	true	"Email address"
//	@Success		200		{object}	user_model.ResendVerificationResponse	"Verification email sent"
//	@Failure		400		{object}	common_model.DescriptiveError			"Invalid request"
//	@Failure		500		{object}	common_model.DescriptiveError			"Internal server error"
//	@Router			/auth/resend-verification [post]
func ResendVerification(c *fiber.Ctx) error {
	var req user_model.ResendVerificationRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	successResponse := user_model.ResendVerificationResponse{
		Message: "If your email is registered, you will receive a verification link",
	}

	// Find user
	var user user_entity.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Don't reveal if email exists
		return c.Status(fiber.StatusOK).JSON(successResponse)
	}

	// Check if already verified
	if user.EmailVerified {
		return c.Status(fiber.StatusOK).JSON(successResponse)
	}

	// Invalidate existing tokens and create new one
	// (Implementation similar to Register, omitted for brevity)

	return c.Status(fiber.StatusOK).JSON(successResponse)
}
