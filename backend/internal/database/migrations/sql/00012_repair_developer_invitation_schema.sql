-- +goose Up
-- Repair databases where migration 00010 was recorded as applied without all
-- of its invitation columns being present.
ALTER TABLE developer_merchant_access
    ADD COLUMN IF NOT EXISTS invitation_token_hash TEXT,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

UPDATE developer_merchant_access
SET invitation_token_hash = id,
    expires_at = created_at + INTERVAL '7 days'
WHERE invitation_token_hash IS NULL;

ALTER TABLE developer_merchant_access
    ALTER COLUMN invitation_token_hash SET NOT NULL,
    ALTER COLUMN developer_id DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_developer_merchant_access_invitation_token
    ON developer_merchant_access (invitation_token_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_developer_merchant_access_invitation_token;
ALTER TABLE developer_merchant_access
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS invitation_token_hash;
