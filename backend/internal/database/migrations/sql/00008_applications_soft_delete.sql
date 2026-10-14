-- +goose Up
ALTER TABLE applications ADD COLUMN deleted_at TIMESTAMPTZ;
CREATE INDEX idx_applications_merchant_deleted_at ON applications (merchant_id, deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_applications_merchant_deleted_at;
ALTER TABLE applications DROP COLUMN deleted_at;
