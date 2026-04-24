package workspace_handler

import (
	"strings"
	"time"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	crypto_service "github.com/Astervia/wacraft-core/src/crypto/service"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	"github.com/Astervia/wacraft-server/src/database"
	email_service "github.com/Astervia/wacraft-server/src/email/service"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
)

// CreateInvitation creates a workspace invitation.
//
//	@Summary		Invite user to workspace
//	@Description	Creates an invitation for a user to join the workspace. Always returns the invitation token so the inviter can share it out-of-band when email is not configured.
//	@Tags			Workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path		string								true	"Workspace ID"
//	@Param			body			body		workspace_model.CreateInvitationRequest	true	"Invitation data"
//	@Success		201				{object}	workspace_model.InvitationResponse	"Invitation created"
//	@Failure		400				{object}	common_model.DescriptiveError		"Invalid request"
//	@Failure		409				{object}	common_model.DescriptiveError		"User already member or pending invite"
//	@Failure		500				{object}	common_model.DescriptiveError		"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace/{workspace_id}/invitation [post]
func CreateInvitation(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)
	user := workspace_middleware.GetUser(c)

	var req workspace_model.CreateInvitationRequest
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

	// Validate policies
	for _, policy := range req.Policies {
		if !workspace_model.IsValidPolicy(policy) {
			return c.Status(fiber.StatusBadRequest).JSON(
				common_model.NewApiError("Invalid policy: "+string(policy), nil, "validation").Send(),
			)
		}
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check if user is already a member
	var existingMember workspace_entity.WorkspaceMember
	if err := database.DB.
		Joins("JOIN users ON users.id = workspace_members.user_id").
		Where("workspace_members.workspace_id = ? AND LOWER(users.email) = ?", workspace.ID, email).
		First(&existingMember).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(
			common_model.NewApiError("User is already a member of this workspace", nil, "workspace").Send(),
		)
	}

	// Check for existing pending invitation
	var existingInvitation workspace_entity.WorkspaceInvitation
	if err := database.DB.Where("workspace_id = ? AND email = ? AND accepted_at IS NULL AND expires_at > ?",
		workspace.ID, email, time.Now()).First(&existingInvitation).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(
			common_model.NewApiError("An invitation is already pending for this email", nil, "workspace").Send(),
		)
	}

	// Generate invitation token
	token, err := crypto_service.GenerateInvitationToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to generate invitation token", err, "crypto").Send(),
		)
	}

	invitation := workspace_entity.WorkspaceInvitation{
		WorkspaceID: workspace.ID,
		Email:       email,
		Token:       token,
		Policies:    req.Policies,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour), // 7 days
		InvitedBy:   user.ID,
	}

	if err := database.DB.Create(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to create invitation", err, "database").Send(),
		)
	}

	// Attempt email send (async; logs only when SMTP is not configured)
	origin := strings.TrimRight(c.Get("Origin"), "/")
	go func() {
		if err := email_service.DefaultEmailService.SendWorkspaceInvitation(
			email, workspace.Name, user.Name, token, origin,
		); err != nil {
			_ = err
		}
	}()

	return c.Status(fiber.StatusCreated).JSON(workspace_model.InvitationResponse{
		ID:          invitation.ID.String(),
		WorkspaceID: workspace.ID.String(),
		Email:       email,
		Token:       token,
		Policies:    req.Policies,
		ExpiresAt:   invitation.ExpiresAt.Format(time.RFC3339),
		InvitedBy:   user.ID.String(),
	})
}

// GetInvitations lists pending invitations for a workspace.
//
//	@Summary		List workspace invitations
//	@Description	Returns a list of pending invitations for the workspace.
//	@Tags			Workspace
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Success		200				{array}		workspace_model.InvitationResponse	"Invitations"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace/{workspace_id}/invitation [get]
func GetInvitations(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)

	var invitations []workspace_entity.WorkspaceInvitation
	if err := database.DB.Where("workspace_id = ? AND accepted_at IS NULL", workspace.ID).
		Preload("Inviter").Find(&invitations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to get invitations", err, "database").Send(),
		)
	}

	responses := make([]workspace_model.InvitationResponse, len(invitations))
	for i, inv := range invitations {
		responses[i] = workspace_model.InvitationResponse{
			ID:          inv.ID.String(),
			WorkspaceID: workspace.ID.String(),
			Email:       inv.Email,
			Token:       inv.Token,
			Policies:    inv.Policies,
			ExpiresAt:   inv.ExpiresAt.Format(time.RFC3339),
			InvitedBy:   inv.InvitedBy.String(),
		}
	}

	return c.Status(fiber.StatusOK).JSON(responses)
}

// RevokeInvitation revokes a pending invitation.
//
//	@Summary		Revoke invitation
//	@Description	Revokes a pending workspace invitation.
//	@Tags			Workspace
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Param			invitation_id	path		string							true	"Invitation ID"
//	@Success		204				{object}	nil								"Invitation revoked"
//	@Failure		404				{object}	common_model.DescriptiveError	"Invitation not found"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace/{workspace_id}/invitation/{invitation_id} [delete]
func RevokeInvitation(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)
	invitationID := c.Params("invitation_id")

	result := database.DB.Where("id = ? AND workspace_id = ? AND accepted_at IS NULL",
		invitationID, workspace.ID).Delete(&workspace_entity.WorkspaceInvitation{})

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to revoke invitation", result.Error, "database").Send(),
		)
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Invitation not found", nil, "workspace").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ClaimInvitation claims a workspace invitation for the currently authenticated user.
// The authenticated user's email must match the invitation email exactly (case-insensitive).
//
//	@Summary		Claim invitation
//	@Description	Claims a workspace invitation. The caller must be authenticated and their email must match the invited email. On success the user is added as a workspace member with the invited policies.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		workspace_model.ClaimInvitationRequest		true	"Claim data"
//	@Success		200		{object}	workspace_model.ClaimInvitationResponse		"Invitation claimed"
//	@Failure		400		{object}	common_model.DescriptiveError	"Invalid or expired invitation"
//	@Failure		403		{object}	common_model.DescriptiveError	"Email mismatch"
//	@Failure		409		{object}	common_model.DescriptiveError	"Already a member"
//	@Failure		500		{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/auth/invitation/claim [post]
func ClaimInvitation(c *fiber.Ctx) error {
	user := workspace_middleware.GetUser(c)

	var req workspace_model.ClaimInvitationRequest
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

	// Find invitation by token
	var invitation workspace_entity.WorkspaceInvitation
	if err := database.DB.Where("token = ?", req.Token).
		Preload("Workspace").First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid invitation token", err, "auth").Send(),
		)
	}

	// Validate invitation state
	if !invitation.IsValid() {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invitation is invalid or expired", nil, "auth").Send(),
		)
	}

	// Enforce email match (case-insensitive)
	if strings.ToLower(strings.TrimSpace(user.Email)) != strings.ToLower(strings.TrimSpace(invitation.Email)) {
		return c.Status(fiber.StatusForbidden).JSON(
			common_model.NewApiError("This invitation was not issued for your email address", nil, "auth").Send(),
		)
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to start transaction", tx.Error, "database").Send(),
		)
	}

	// Check if already a member
	var existingMember workspace_entity.WorkspaceMember
	if err := tx.Where("workspace_id = ? AND user_id = ?", invitation.WorkspaceID, user.ID).First(&existingMember).Error; err == nil {
		tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(
			common_model.NewApiError("User is already a member of this workspace", nil, "workspace").Send(),
		)
	}

	// Add user as workspace member
	member := workspace_entity.WorkspaceMember{
		WorkspaceID: invitation.WorkspaceID,
		UserID:      user.ID,
	}

	if err := tx.Create(&member).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to add member", err, "database").Send(),
		)
	}

	// Assign policies from invitation in batch
	if len(invitation.Policies) > 0 {
		// ⚡ BOLT OPTIMIZATION: Use GORM batch insert to reduce database round-trips
		// Expected impact: Faster database operations and reduced transaction overhead
		policiesToInsert := make([]workspace_entity.WorkspaceMemberPolicy, 0, len(invitation.Policies))
		for _, policy := range invitation.Policies {
			policiesToInsert = append(policiesToInsert, workspace_entity.WorkspaceMemberPolicy{
				WorkspaceMemberID: member.ID,
				Policy:            policy,
			})
		}

		if err := tx.Create(&policiesToInsert).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("Failed to assign policies in batch", err, "database").Send(),
			)
		}
	}

	// Mark invitation as accepted
	now := time.Now()
	invitation.AcceptedAt = &now
	if err := tx.Save(&invitation).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to update invitation", err, "database").Send(),
		)
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Failed to commit transaction", err, "database").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(workspace_model.ClaimInvitationResponse{
		Message:     "Invitation claimed",
		WorkspaceID: invitation.WorkspaceID.String(),
	})
}
