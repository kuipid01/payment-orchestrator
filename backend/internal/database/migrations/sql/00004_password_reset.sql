-- +goose Up
ALTER TABLE users
    ADD COLUMN password_reset_token_hash TEXT,
    ADD COLUMN password_reset_expires_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_password_reset_token_hash
    ON users (password_reset_token_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_users_password_reset_token_hash;
ALTER TABLE users
    DROP COLUMN password_reset_token_hash,
    DROP COLUMN password_reset_expires_at;
