-- +goose Up
ALTER TABLE users
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN email_verification_token_hash TEXT UNIQUE,
    ADD COLUMN email_verification_expires_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users
    DROP COLUMN email_verification_expires_at,
    DROP COLUMN email_verification_token_hash,
    DROP COLUMN email_verified;