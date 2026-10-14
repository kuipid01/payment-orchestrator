-- +goose Up
ALTER TABLE developer_merchant_access
    ALTER COLUMN developer_id DROP NOT NULL;

-- +goose Down
ALTER TABLE developer_merchant_access
    ALTER COLUMN developer_id SET NOT NULL;
