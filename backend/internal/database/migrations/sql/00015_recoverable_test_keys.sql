-- +goose Up
-- Test keys stay recoverable; live keys do not.
--
-- A test key cannot move real money: its environment is pinned to "test" and the
-- authenticator refuses to let it read or write live data. But it is the
-- credential a developer needs on every run, runbook and CI job, and forcing a
-- rotation just to re-read it churns integration config for no security gain.
-- So a test key's plaintext is kept encrypted with the platform vault key and can
-- be shown again to an authenticated merchant session.
--
-- Live keys keep the show-once behaviour: only their SHA-256 hash is stored, and
-- NULL here means "not recoverable", whether because the key is live or because
-- it predates this migration.
ALTER TABLE api_keys ADD COLUMN secret_encrypted BYTEA;

-- +goose Down
ALTER TABLE api_keys DROP COLUMN secret_encrypted;
