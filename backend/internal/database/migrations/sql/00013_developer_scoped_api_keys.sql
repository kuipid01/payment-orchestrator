-- +goose Up
-- A developer-scoped key is bound to one developer grant, so revoking that
-- person's access kills their keys without rotating the application's shared
-- key. NULL means a general application key the team shares.
--
-- The grant -- not the developer identity -- is the reference because a key can
-- be issued while the invitation is still pending; the grant's id is stable
-- across acceptance and only becomes usable once the grant is active.
ALTER TABLE api_keys
    ADD COLUMN developer_access_id TEXT REFERENCES developer_merchant_access(id) ON DELETE SET NULL;

CREATE INDEX idx_api_keys_developer_access ON api_keys (developer_access_id);

-- +goose Down
DROP INDEX IF EXISTS idx_api_keys_developer_access;
ALTER TABLE api_keys DROP COLUMN developer_access_id;
