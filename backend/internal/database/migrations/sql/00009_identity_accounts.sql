-- +goose Up
-- Split dashboard identity from merchant membership. Existing users remain
-- valid by being copied into user_merchants, and their original merchant is
-- retained as the first/default membership.
ALTER TABLE merchants
    ADD COLUMN owner_user_id TEXT;

UPDATE merchants
SET owner_user_id = (
    SELECT u.id
    FROM users u
    WHERE u.merchant_id = merchants.id
    ORDER BY u.created_at
    LIMIT 1
)
WHERE owner_user_id IS NULL;

ALTER TABLE merchants
    ALTER COLUMN owner_user_id SET NOT NULL;

CREATE TABLE user_merchants (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    merchant_id TEXT NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'Owner',
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, merchant_id)
);

INSERT INTO user_merchants (id, user_id, merchant_id, role, status, created_at, updated_at)
SELECT
    'um_' || lpad(to_char(row_number() OVER (ORDER BY u.id), 'FM999999999999999999'), 18, '0'),
    u.id,
    u.merchant_id,
    COALESCE(u.role, 'Owner'),
    COALESCE(u.status, 'active'),
    u.created_at,
    now()
FROM users u
WHERE u.merchant_id IS NOT NULL;

ALTER TABLE users
    DROP COLUMN merchant_id,
    DROP COLUMN role,
    DROP COLUMN is_developer;

CREATE TABLE developers (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE developer_merchant_access (
    id          TEXT PRIMARY KEY,
    developer_id TEXT NOT NULL REFERENCES developers(id) ON DELETE CASCADE,
    merchant_id  TEXT NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    invitee_email TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    permissions JSONB NOT NULL DEFAULT '[]',
    invited_by  TEXT NOT NULL REFERENCES users(id),
    accepted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (developer_id, merchant_id)
);

-- Postgres rejects expressions in table constraints, so the case-insensitive
-- uniqueness of an invitee e-mail is enforced with a dedicated unique index.
CREATE UNIQUE INDEX uniq_developer_merchant_access_email
    ON developer_merchant_access (merchant_id, lower(invitee_email));

CREATE INDEX idx_user_merchants_user ON user_merchants(user_id, status);
CREATE INDEX idx_user_merchants_merchant ON user_merchants(merchant_id, status);
CREATE INDEX idx_developer_merchant_access_developer ON developer_merchant_access(developer_id, status);
CREATE INDEX idx_developer_merchant_access_merchant ON developer_merchant_access(merchant_id, status);

-- +goose Down
ALTER TABLE users
    ADD COLUMN merchant_id TEXT,
    ADD COLUMN role TEXT NOT NULL DEFAULT 'Owner',
    ADD COLUMN is_developer BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users u
SET merchant_id = first_membership.merchant_id
FROM (
    SELECT DISTINCT ON (user_id) user_id, merchant_id
    FROM user_merchants
    ORDER BY user_id, created_at
) first_membership
WHERE first_membership.user_id = u.id;

UPDATE users u
SET is_developer = EXISTS (
    SELECT 1 FROM developers d WHERE d.user_id = u.id
);

DROP INDEX IF EXISTS uniq_developer_merchant_access_email;
DROP TABLE developer_merchant_access;
DROP TABLE developers;
DROP TABLE user_merchants;

ALTER TABLE merchants DROP COLUMN owner_user_id;
