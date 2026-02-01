package workspace_handler

import (
	"regexp"
	"strings"

	common_model "github.com/Astervia/wacraft-core/src/common/model"
	"github.com/Astervia/wacraft-core/src/repository"
	user_entity "github.com/Astervia/wacraft-core/src/user/entity"
	workspace_entity "github.com/Astervia/wacraft-core/src/workspace/entity"
	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-server/src/validators"
	workspace_middleware "github.com/Astervia/wacraft-server/src/workspace/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Create creates a new workspace.
//
//	@Summary		Create a new workspace
//	@Description	Creates a new workspace and adds the creator as admin member.
//	@Tags			Workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspace	body		workspace_model.CreateWorkspace		true	"Workspace data"
//	@Success		201			{object}	workspace_entity.Workspace			"Created workspace"
//	@Failure		400			{object}	common_model.DescriptiveError		"Invalid request body"
//	@Failure		500			{object}	common_model.DescriptiveError		"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace [post]
func Create(c *fiber.Ctx) error {
	var newWorkspace workspace_model.CreateWorkspace
	if err := c.BodyParser(&newWorkspace); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	if err := validators.Validator().Struct(&newWorkspace); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewValidationError(err).Send(),
		)
	}

	// Normalize slug
	newWorkspace.Slug = normalizeSlug(newWorkspace.Slug)

	// Get current user
	user, ok := c.Locals("user").(*user_entity.User)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("User not authenticated", nil, "handler").Send(),
		)
	}

	// Start transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create workspace
	workspace, err := repository.Create(
		workspace_entity.Workspace{
			Name:        newWorkspace.Name,
			Slug:        newWorkspace.Slug,
			Description: newWorkspace.Description,
			CreatedBy:   user.ID,
		}, tx,
	)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to create workspace", err, "repository").Send(),
		)
	}

	// Create workspace member for the creator
	member, err := repository.Create(
		workspace_entity.WorkspaceMember{
			WorkspaceID: workspace.ID,
			UserID:      user.ID,
		}, tx,
	)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to create workspace member", err, "repository").Send(),
		)
	}

	// Add all policies to the creator (workspace admin)
	for _, policy := range workspace_model.AllPolicies {
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

	return c.Status(fiber.StatusCreated).JSON(workspace)
}

// Get lists all workspaces the user is a member of.
//
//	@Summary		List user's workspaces
//	@Description	Returns a list of all workspaces the authenticated user is a member of.
//	@Tags			Workspace
//	@Produce		json
//	@Success		200		{array}		workspace_entity.Workspace			"List of workspaces"
//	@Failure		500		{object}	common_model.DescriptiveError		"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace [get]
func Get(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*user_entity.User)
	if !ok || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			common_model.NewApiError("User not authenticated", nil, "handler").Send(),
		)
	}

	var workspaces []workspace_entity.Workspace
	err := database.DB.
		Joins("JOIN workspace_members ON workspace_members.workspace_id = workspaces.id").
		Where("workspace_members.user_id = ?", user.ID).
		Find(&workspaces).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to fetch workspaces", err, "database").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(workspaces)
}

// GetByID gets a specific workspace by ID.
//
//	@Summary		Get workspace by ID
//	@Description	Returns a specific workspace by its ID.
//	@Tags			Workspace
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Success		200				{object}	workspace_entity.Workspace		"Workspace details"
//	@Failure		404				{object}	common_model.DescriptiveError	"Workspace not found"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace/{workspace_id} [get]
func GetByID(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)
	if workspace == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Workspace not found", nil, "handler").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(workspace)
}

// Update updates a workspace.
//
//	@Summary		Update workspace
//	@Description	Updates an existing workspace. Requires workspace.settings policy.
//	@Tags			Workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspace_id	path		string								true	"Workspace ID"
//	@Param			workspace		body		workspace_model.UpdateWorkspace		true	"Updated workspace data"
//	@Success		200				{object}	workspace_entity.Workspace			"Updated workspace"
//	@Failure		400				{object}	common_model.DescriptiveError		"Invalid request body"
//	@Failure		403				{object}	common_model.DescriptiveError		"Forbidden"
//	@Failure		500				{object}	common_model.DescriptiveError		"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace/{workspace_id} [patch]
func Update(c *fiber.Ctx) error {
	workspace := workspace_middleware.GetWorkspace(c)
	if workspace == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			common_model.NewApiError("Workspace not found", nil, "handler").Send(),
		)
	}

	var updateData workspace_model.UpdateWorkspace
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewParseJsonError(err).Send(),
		)
	}

	// Build update map
	updates := make(map[string]interface{})
	if updateData.Name != nil {
		updates["name"] = *updateData.Name
	}
	if updateData.Slug != nil {
		updates["slug"] = normalizeSlug(*updateData.Slug)
	}
	if updateData.Description != nil {
		updates["description"] = *updateData.Description
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusOK).JSON(workspace)
	}

	if err := database.DB.Model(workspace).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to update workspace", err, "database").Send(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(workspace)
}

// Delete deletes a workspace.
//
//	@Summary		Delete workspace
//	@Description	Deletes a workspace. Requires workspace.admin policy.
//	@Tags			Workspace
//	@Produce		json
//	@Param			workspace_id	path		string							true	"Workspace ID"
//	@Success		204				{object}	nil								"Workspace deleted"
//	@Failure		403				{object}	common_model.DescriptiveError	"Forbidden"
//	@Failure		500				{object}	common_model.DescriptiveError	"Internal server error"
//	@Security		ApiKeyAuth
//	@Security		WorkspaceAuth
//	@Router			/workspace/{workspace_id} [delete]
func Delete(c *fiber.Ctx) error {
	workspaceIDStr := c.Params("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			common_model.NewApiError("Invalid workspace ID", err, "handler").Send(),
		)
	}

	if err := repository.DeleteByID[workspace_entity.Workspace](workspaceID, database.DB); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			common_model.NewApiError("Unable to delete workspace", err, "repository").Send(),
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// normalizeSlug converts a string to a URL-safe slug
func normalizeSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile("[^a-z0-9-]+")
	s = reg.ReplaceAllString(s, "")
	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile("-+")
	s = reg.ReplaceAllString(s, "-")
	// Trim hyphens from start and end
	s = strings.Trim(s, "-")
	return s
}
