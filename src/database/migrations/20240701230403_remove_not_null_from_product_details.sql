-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
DO $$
BEGIN
    -- Check if the column 'product_details' in 'messaging_product_contacts' table has a NOT NULL constraint
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name='messaging_product_contacts' 
          AND column_name='product_details' 
          AND is_nullable = 'NO'
    ) THEN
        -- Drop the NOT NULL constraint
        ALTER TABLE messaging_product_contacts ALTER COLUMN product_details DROP NOT NULL;
    END IF;
END$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE messaging_product_contacts ALTER COLUMN product_details SET NOT NULL;
-- +goose StatementEnd
