-- +goose Up
-- Fixups discovered while reconnecting the domains to the schema:
--  * provider_connections stored its ciphertext in `secret_encrypted`, but the
--    domain model reads/writes `credentials_encrypted`.
--  * api_logs had no request_id column, so request correlation could not be
--    persisted.
--  * webhook_endpoints required an application, which the merchant-level
--    endpoints in the specification do not have.
--  * payments.reference was globally unique; references are merchant-scoped.

ALTER TABLE provider_connections RENAME COLUMN secret_encrypted TO credentials_encrypted;

ALTER TABLE api_logs ADD COLUMN request_id TEXT NOT NULL DEFAULT '';
ALTER TABLE api_logs DROP CONSTRAINT IF EXISTS api_logs_application_id_fkey;
ALTER TABLE api_logs ALTER COLUMN application_id DROP NOT NULL;

ALTER TABLE webhook_endpoints ALTER COLUMN application_id DROP NOT NULL;
ALTER TABLE webhook_endpoints DROP COLUMN secret_new;

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_reference_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_merchant_env_reference
    ON payments (merchant_id, environment, reference);

-- +goose Down
DROP INDEX IF EXISTS idx_payments_merchant_env_reference;
ALTER TABLE payments ADD CONSTRAINT payments_reference_key UNIQUE (reference);

ALTER TABLE webhook_endpoints ADD COLUMN secret_new TEXT;
ALTER TABLE webhook_endpoints ALTER COLUMN application_id SET NOT NULL;

ALTER TABLE api_logs ALTER COLUMN application_id SET NOT NULL;
ALTER TABLE api_logs ADD CONSTRAINT api_logs_application_id_fkey
    FOREIGN KEY (application_id) REFERENCES applications(id);
ALTER TABLE api_logs DROP COLUMN request_id;

ALTER TABLE provider_connections RENAME COLUMN credentials_encrypted TO secret_encrypted;
