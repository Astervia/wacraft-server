package workspace_handler

import (
	common_model "github.com/Astervia/wacraft-core/src/common/model"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	"github.com/Astervia/wacraft-core/src/repository"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// MemberResponse represents a workspace member with their policies
type MemberResponse struct {
	workspace_entity.WorkspaceMember
	Policies []workspace_model.Policy `json:"policies"`
}

// AddMember adds a new member to a workspace.
//
//	@Summary		Add workspace member
//	@Description	Adds a new member to the workspace with specified policies. Requires workspace.members policy.
//	@Tags			Workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path		string								true	"Workspace ID"
//	@Param			member			body		workspace_model.CreateMember		true	"Member data"
//	@Success		201				{object}	MemberResponse						"Created member"
//	@Failure		400				{object}	common_model.DescriptiveError		"Invalid request body"
//	@Failure		403				{object}	common_model.DescriptiveError		"Forbidden"
//	@Failure		500				{object}	common_model.DescriptiveError		"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/workspace/{workspace_id}/member [post]
func AddMember(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)
	if workspace == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Workspace not found", nil, "handler").Send(),
		)
	}

	var newMember workspace_model.CreateMember
	if err := c.BodyParser(&newMember); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&newMember); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	// Validate policies
	for _, policy := range newMember.Policies {
		if !workspace_model.IsValidPolicy(policy) {
			return c.Status(fiber.StatusBadRequest).JSON(
				common_model.NewApiError("Invalid policy: "+string(policy), nil, "handler").Send(),
			)
		}
	}

	// Check if user is already a member
	var existingMember workspace_entity.WorkspaceMember
	err := database.DB.Where("workspace_id = ? AND user_id = ?", workspace.ID, newMember.UserID).First(&existingMember).Error
	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(
			common_model.NewApiError("User is already a member of this workspace", nil, "handler").Send(),
		)
	}

	// Start transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create workspace member
	member, err := repository.Create(
		workspace_entity.WorkspaceMember{
			WorkspaceID: workspace.ID,
			UserID:      newMember.UserID,
		}, tx,
	)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to create workspace member", err, "repository").Send(),
		)
	}

	// Add policies
	for _, policy := range newMember.Policies {
		_, err := repository.Create(
			workspace_entity.WorkspaceMemberPolicy{
				WorkspaceMemberID: member.ID,
				Policy:            policy,
			}, tx,
		)
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("Unable to create workspace policy", err, "repository").Send(),
			)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to commit transaction", err, "database").Send(),
		)
	}

	response := MemberResponse{
		WorkspaceMember: member,
		Policies:        newMember.Policies,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// GetMembers lists all members of a workspace.
//
//	@Summary		List workspace members
//	@Description	Returns a list of all members in the workspace with their policies.
//	@Tags			Workspace
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Success		200				{array}		MemberResponse					"List of members"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/workspace/{workspace_id}/member [get]
func GetMembers(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)
	if workspace == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Workspace not found", nil, "handler").Send(),
		)
	}

	var members []workspace_entity.WorkspaceMember
	if err := database.DB.Preload("User").Where("workspace_id = ?", workspace.ID).Find(&members).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to fetch workspace members", err, "database").Send(),
		)
	}

	// Fetch policies for each member
	responses := make([]MemberResponse, len(members))
	for i, member := range members {
		var policies []workspace_entity.WorkspaceMemberPolicy
		if err := database.DB.Where("workspace_member_id = ?", member.ID).Find(&policies).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("Unable to fetch member policies", err, "database").Send(),
			)
		}

		policyStrings := make([]workspace_model.Policy, len(policies))
		for j, p := range policies {
			policyStrings[j] = p.Policy
		}

		responses[i] = MemberResponse{
			WorkspaceMember: member,
			Policies:        policyStrings,
		}
	}

	return c.Status(fiber.StatusOK).JSON(responses)
}

// UpdateMemberPolicies updates a member's policies.
//
//	@Summary		Update member policies
//	@Description	Updates the policies for a workspace member. Requires workspace.members policy.
//	@Tags			Workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path		string									true	"Workspace ID"
//	@Param			user_id			path		string									true	"User ID"
//	@Param			policies		body		workspace_model.UpdateMemberPolicies	true	"New policies"
//	@Success		200				{object}	MemberResponse							"Updated member"
//	@Failure		400				{object}	common_model.DescriptiveError			"Invalid request body"
//	@Failure		403				{object}	common_model.DescriptiveError			"Forbidden"
//	@Failure		404				{object}	common_model.DescriptiveError			"Member not found"
//	@Failure		500				{object}	common_model.DescriptiveError			"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/workspace/{workspace_id}/member/{user_id} [patch]
func UpdateMemberPolicies(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)
	if workspace == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Workspace not found", nil, "handler").Send(),
		)
	}

	userIDStr := c.Params("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid user ID", err, "handler").Send(),
		)
	}

	var updateData workspace_model.UpdateMemberPolicies
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	// Validate policies
	for _, policy := range updateData.Policies {
		if !workspace_model.IsValidPolicy(policy) {
			return c.Status(fiber.StatusBadRequest).JSON(
				common_model.NewApiError("Invalid policy: "+string(policy), nil, "handler").Send(),
			)
		}
	}

	// Find the member
	var member workspace_entity.WorkspaceMember
	if err := database.DB.Where("workspace_id = ? AND user_id = ?", workspace.ID, userID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Member not found", err, "handler").Send(),
		)
	}

	// Start transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete existing policies
	if err := tx.Where("workspace_member_id = ?", member.ID).Delete(&workspace_entity.WorkspaceMemberPolicy{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to delete existing policies", err, "database").Send(),
		)
	}

	// Add new policies
	for _, policy := range updateData.Policies {
		_, err := repository.Create(
			workspace_entity.WorkspaceMemberPolicy{
				WorkspaceMemberID: member.ID,
				Policy:            policy,
			}, tx,
		)
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(
				common_model.NewApiError("Unable to create workspace policy", err, "repository").Send(),
			)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to commit transaction", err, "database").Send(),
		)
	}

	response := MemberResponse{
		WorkspaceMember: member,
		Policies:        updateData.Policies,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// RemoveMember removes a member from a workspace.
//
//	@Summary		Remove workspace member
//	@Description	Removes a member from the workspace. Requires workspace.members policy.
//	@Tags			Workspace
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Param			user_id			path		string							true	"User ID"
//	@Success		204				{object}	nil								"Member removed"
//	@Failure		403				{object}	common_model.DescriptiveError	"Forbidden"
//	@Failure		404				{object}	common_model.DescriptiveError	"Member not found"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/workspace/{workspace_id}/member/{user_id} [delete]
func RemoveMember(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)
	if workspace == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Workspace not found", nil, "handler").Send(),
		)
	}

	userIDStr := c.Params("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid user ID", err, "handler").Send(),
		)
	}

	// Find and delete the member
	result := database.DB.Where("workspace_id = ? AND user_id = ?", workspace.ID, userID).Delete(&workspace_entity.WorkspaceMember{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to remove member", result.Error, "database").Send(),
		)
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Member not found", nil, "handler").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
