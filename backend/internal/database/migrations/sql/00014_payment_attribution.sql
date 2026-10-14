-- +goose Up
-- Attribute each payment to the credential that created it.
--
-- Until now a payment knew its merchant and (optionally) its application, so the
-- dashboard could not answer "which developer took this payment". A payment is
-- created with an API key, and a key is either general (shared by the team) or
-- developer-scoped; recording both the key and the grant behind it makes the
-- listing filterable per developer without ever re-deriving it from a key that
-- may since have been revoked.
--
-- Both columns are nullable: rows written before this migration, and payments
-- that never had a key behind them, keep NULL. The foreign keys use ON DELETE SET
-- NULL so deleting a key or a grant never removes money history.
ALTER TABLE payments
    ADD COLUMN api_key_id TEXT REFERENCES api_keys(id) ON DELETE SET NULL,
    ADD COLUMN developer_access_id TEXT REFERENCES developer_merchant_access(id) ON DELETE SET NULL;

CREATE INDEX idx_payments_application ON payments (application_id, created_at DESC);
CREATE INDEX idx_payments_developer_access ON payments (developer_access_id, created_at DESC);
CREATE INDEX idx_payments_api_key ON payments (api_key_id);

-- +goose Down
DROP INDEX IF EXISTS idx_payments_api_key;
DROP INDEX IF EXISTS idx_payments_developer_access;
DROP INDEX IF EXISTS idx_payments_application;

ALTER TABLE payments DROP COLUMN developer_access_id;
ALTER TABLE payments DROP COLUMN api_key_id;
