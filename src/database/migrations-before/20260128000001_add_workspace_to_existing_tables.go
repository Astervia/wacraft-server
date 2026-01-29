package migrations

import (
	"context"
	"database/sql"
	"fmt"

	workspace_model "github.com/Astervia/wacraft-core/src/workspace/model"
	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
)

func init() {
	goose.AddMigrationContext(upAddWorkspaceToExistingTables, downAddWorkspaceToExistingTables)
}

func tableExists(ctx context.Context, tx *sql.Tx, tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_name = $1
		)
	`
	err := tx.QueryRowContext(ctx, query, tableName).Scan(&exists)
	return exists, err
}

func columnExists(ctx context.Context, tx *sql.Tx, tableName, columnName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_name = $1
			AND column_name = $2
		)
	`
	err := tx.QueryRowContext(ctx, query, tableName, columnName).Scan(&exists)
	return exists, err
}

func upAddWorkspaceToExistingTables(ctx context.Context, tx *sql.Tx) error {
	pterm.DefaultLogger.Info("Running workspace migration for existing tables...")

	// Check if contacts table exists (indicates existing DB vs virgin DB)
	contactsExists, err := tableExists(ctx, tx, "contacts")
	if err != nil {
		return err
	}

	if !contactsExists {
		pterm.DefaultLogger.Info("Virgin database detected, skipping migration (AutoMigrate will handle everything)")
		return nil
	}

	pterm.DefaultLogger.Info("Existing database detected, migrating to workspace structure...")

	// First, create workspace tables if they don't exist
	workspacesExists, err := tableExists(ctx, tx, "workspaces")
	if err != nil {
		return err
	}

	if !workspacesExists {
		pterm.DefaultLogger.Info("Creating workspace tables...")

		// Create workspaces table
		_, err = tx.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS workspaces (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				name VARCHAR(255) NOT NULL,
				slug VARCHAR(255) NOT NULL,
				description TEXT,
				created_by UUID NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				CONSTRAINT uni_workspaces_slug UNIQUE (slug)
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create workspaces table: %w", err)
		}

		// Create workspace_members table
		_, err = tx.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS workspace_members (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				workspace_id UUID NOT NULL REFERENCES workspaces(id) ON UPDATE CASCADE ON DELETE CASCADE,
				user_id UUID NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				UNIQUE(workspace_id, user_id)
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create workspace_members table: %w", err)
		}

		// Create workspace_member_policies table
		_, err = tx.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS workspace_member_policies (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				workspace_member_id UUID NOT NULL REFERENCES workspace_members(id) ON UPDATE CASCADE ON DELETE CASCADE,
				policy VARCHAR(50) NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create workspace_member_policies table: %w", err)
		}

		// Create indexes
		_, _ = tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace_id ON workspace_members(workspace_id)`)
		_, _ = tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_workspace_members_user_id ON workspace_members(user_id)`)
		_, _ = tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_workspace_member_policies_member_id ON workspace_member_policies(workspace_member_id)`)
	}

	// Get the super user's ID
	var suUserID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = 'su@sudo' LIMIT 1`).Scan(&suUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			pterm.DefaultLogger.Warn("No super user found, skipping workspace data migration")
			return nil
		}
		return err
	}

	// Check if default workspace already exists
	var workspaceID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM workspaces WHERE slug = 'default' LIMIT 1`).Scan(&workspaceID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Create default workspace if it doesn't exist
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO workspaces (name, slug, description, created_by)
			VALUES ('Default Workspace', 'default', 'Default workspace for existing data', $1)
			RETURNING id
		`, suUserID).Scan(&workspaceID)
		if err != nil {
			return fmt.Errorf("failed to create default workspace: %w", err)
		}
		pterm.DefaultLogger.Info("Created default workspace: " + workspaceID)

		// Create workspace member for super user
		var memberID string
		err = tx.QueryRowContext(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id)
			VALUES ($1, $2)
			RETURNING id
		`, workspaceID, suUserID).Scan(&memberID)
		if err != nil {
			return fmt.Errorf("failed to create workspace member: %w", err)
		}
		pterm.DefaultLogger.Info("Added super user as workspace member")

		// Add all policies to the super user
		for _, policy := range workspace_model.AllPolicies {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO workspace_member_policies (workspace_member_id, policy)
				VALUES ($1, $2)
			`, memberID, string(policy))
			if err != nil {
				return fmt.Errorf("failed to add policy %s: %w", policy, err)
			}
		}
		pterm.DefaultLogger.Info("Added all policies to super user")
	}

	// Add workspace_id column to existing tables if not present
	tables := []string{"contacts", "messaging_products", "campaigns", "webhooks"}
	for _, table := range tables {
		exists, err := tableExists(ctx, tx, table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		colExists, err := columnExists(ctx, tx, table, "workspace_id")
		if err != nil {
			return err
		}

		if !colExists {
			// Add nullable workspace_id column
			_, err = tx.ExecContext(ctx, fmt.Sprintf(`
				ALTER TABLE %s ADD COLUMN workspace_id UUID
			`, table))
			if err != nil {
				return fmt.Errorf("failed to add workspace_id to %s: %w", table, err)
			}
			pterm.DefaultLogger.Info(fmt.Sprintf("Added workspace_id column to %s", table))
		}

		// Update existing records with the default workspace ID
		result, err := tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE %s SET workspace_id = $1 WHERE workspace_id IS NULL
		`, table), workspaceID)
		if err != nil {
			return fmt.Errorf("failed to update %s with workspace_id: %w", table, err)
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			pterm.DefaultLogger.Info(fmt.Sprintf("Updated %d rows in %s with default workspace", rowsAffected, table))
		}

		// Add index if not exists
		_, _ = tx.ExecContext(ctx, fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS idx_%s_workspace_id ON %s(workspace_id)
		`, table, table))
	}

	// Update campaigns table to have composite unique index on workspace_id and name
	// Only if campaigns table exists (not present in light version)
	campaignsExists, err := tableExists(ctx, tx, "campaigns")
	if err != nil {
		return err
	}

	if campaignsExists {
		// First check if the old unique constraint exists
		var constraintExists bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conname = 'campaigns_name_key'
			)
		`).Scan(&constraintExists)
		if err != nil {
			return err
		}

		if constraintExists {
			_, err = tx.ExecContext(ctx, `ALTER TABLE campaigns DROP CONSTRAINT campaigns_name_key`)
			if err != nil {
				pterm.DefaultLogger.Warn(fmt.Sprintf("Could not drop campaigns_name_key constraint: %v", err))
			}
		}

		// Create composite unique index
		_, _ = tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_campaign_name ON campaigns(workspace_id, name)`)
		pterm.DefaultLogger.Info("Updated campaigns table with workspace constraints")
	} else {
		pterm.DefaultLogger.Info("Campaigns table not found (light version), skipping campaign-specific migration")
	}

	pterm.DefaultLogger.Info("Workspace migration completed successfully")
	return nil
}

func downAddWorkspaceToExistingTables(ctx context.Context, tx *sql.Tx) error {
	pterm.DefaultLogger.Info("Reverting workspace migration...")

	// This is a destructive migration - we can't fully revert
	// Just remove the workspace_id columns and drop workspace tables

	tables := []string{"contacts", "messaging_products", "campaigns", "webhooks"}

	for _, table := range tables {
		exists, err := tableExists(ctx, tx, table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		colExists, err := columnExists(ctx, tx, table, "workspace_id")
		if err != nil {
			return err
		}

		if colExists {
			_, _ = tx.ExecContext(ctx, fmt.Sprintf(`DROP INDEX IF EXISTS idx_%s_workspace_id`, table))
			_, _ = tx.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s DROP COLUMN workspace_id`, table))
		}
	}

	// Restore campaigns unique constraint
	_, _ = tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_workspace_campaign_name`)
	_, _ = tx.ExecContext(ctx, `ALTER TABLE campaigns ADD CONSTRAINT campaigns_name_key UNIQUE (name)`)

	// Drop workspace tables (order matters due to FKs)
	_, _ = tx.ExecContext(ctx, `DROP TABLE IF EXISTS workspace_member_policies`)
	_, _ = tx.ExecContext(ctx, `DROP TABLE IF EXISTS workspace_members`)
	_, _ = tx.ExecContext(ctx, `DROP TABLE IF EXISTS workspaces`)

	pterm.DefaultLogger.Info("Workspace migration reverted")
	return nil
}
