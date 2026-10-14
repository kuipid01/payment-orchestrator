-- +goose Up
ALTER TABLE developer_merchant_access
    ADD COLUMN invitation_token_hash TEXT,
    ADD COLUMN expires_at TIMESTAMPTZ;

UPDATE developer_merchant_access
SET invitation_token_hash = id,
    expires_at = created_at + INTERVAL '7 days'
WHERE invitation_token_hash IS NULL;

ALTER TABLE developer_merchant_access
    ALTER COLUMN invitation_token_hash SET NOT NULL;

CREATE UNIQUE INDEX idx_developer_merchant_access_invitation_token
    ON developer_merchant_access (invitation_token_hash);

-- +goose Down
DROP INDEX IF EXISTS developer_merchant_access_invitation_token;
ALTER TABLE developer_merchant_access DROP COLUMN expires_at;
ALTER TABLE developer_merchant_access DROP COLUMN invitation_token_hash;
