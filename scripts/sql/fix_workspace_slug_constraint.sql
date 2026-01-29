-- Fix workspace slug constraint naming to match GORM expectations
-- This script renames the auto-generated constraint to match GORM's naming convention

DO $$
BEGIN
    -- Drop the auto-generated unique constraint if it exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'workspaces_slug_key'
        AND conrelid = 'workspaces'::regclass
    ) THEN
        ALTER TABLE workspaces DROP CONSTRAINT workspaces_slug_key;
        RAISE NOTICE 'Dropped existing constraint workspaces_slug_key';
    END IF;

    -- Create the constraint with GORM's expected name if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uni_workspaces_slug'
        AND conrelid = 'workspaces'::regclass
    ) THEN
        ALTER TABLE workspaces ADD CONSTRAINT uni_workspaces_slug UNIQUE (slug);
        RAISE NOTICE 'Created constraint uni_workspaces_slug';
    ELSE
        RAISE NOTICE 'Constraint uni_workspaces_slug already exists';
    END IF;
END $$;

-- Verify the constraint exists with the correct name
SELECT
    con.conname AS constraint_name,
    con.contype AS constraint_type
FROM pg_constraint con
JOIN pg_class rel ON rel.oid = con.conrelid
WHERE rel.relname = 'workspaces'
AND con.conname = 'uni_workspaces_slug';
