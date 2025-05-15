-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS messaging_product_id uuid;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                   WHERE constraint_type = 'FOREIGN KEY' 
                   AND table_name = 'messages' 
                   AND constraint_name = 'fk_messaging_product_id') THEN
        ALTER TABLE messages
        ADD CONSTRAINT fk_messaging_product_id
        FOREIGN KEY (messaging_product_id)
        REFERENCES messaging_products(id)
        ON UPDATE CASCADE
        ON DELETE SET NULL;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE messages
DROP CONSTRAINT IF EXISTS fk_messaging_product_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE messages
DROP COLUMN IF EXISTS messaging_product_id;
-- +goose StatementEnd
