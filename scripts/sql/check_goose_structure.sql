-- Check the goose_db_version table structure
\d goose_db_version

-- Show all existing migrations
SELECT * FROM goose_db_version ORDER BY id;
