-- Mark the old migration as completed in goose_db_version table
-- This tells goose that this migration has already been executed

-- Only insert if it doesn't already exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM goose_db_version WHERE version_id = 20241011174817
    ) THEN
        INSERT INTO goose_db_version (version_id, is_applied, tstamp)
        VALUES (20241011174817, true, NOW());
        RAISE NOTICE 'Migration 20241011174817 marked as completed';
    ELSE
        RAISE NOTICE 'Migration 20241011174817 already exists in goose_db_version';
    END IF;
END $$;

-- Verify the migration is marked as completed
SELECT * FROM goose_db_version WHERE version_id = 20241011174817;
