-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE organizations (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id             TEXT PRIMARY KEY,
    org_id         TEXT NOT NULL REFERENCES organizations(id),
    name           TEXT NOT NULL,
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    role           TEXT NOT NULL DEFAULT 'Owner', -- Owner | Admin | Developer | Viewer
    is_developer   BOOLEAN NOT NULL DEFAULT FALSE,
    status         TEXT NOT NULL DEFAULT 'active', -- active | invited | revoked
    last_active_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ⛨ vault: provider credentials, encrypted at rest
CREATE TABLE provider_connections (
    id                TEXT PRIMARY KEY,
    org_id            TEXT NOT NULL REFERENCES organizations(id),
    provider          TEXT NOT NULL,               -- paystack | flutterwave | stripe
    status            TEXT NOT NULL DEFAULT 'not_connected',
    environment       TEXT NOT NULL DEFAULT 'test',
    secret_encrypted  BYTEA,                       -- AES-256-GCM ciphertext (never rendered)
    secret_last4      TEXT NOT NULL DEFAULT '',
    last_verified_at  TIMESTAMPTZ,
    connected_at      TIMESTAMPTZ,
    currency          TEXT NOT NULL DEFAULT 'NGN',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, provider, environment)
);

CREATE TABLE applications (
    id            TEXT PRIMARY KEY,
    org_id        TEXT NOT NULL REFERENCES organizations(id),
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    environment   TEXT NOT NULL DEFAULT 'test',
    status        TEXT NOT NULL DEFAULT 'active',
    last_used_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- developer-facing keys; full key returned only at creation/rotation
CREATE TABLE api_keys (
    id              TEXT PRIMARY KEY,
    org_id          TEXT NOT NULL REFERENCES organizations(id),
    application_id  TEXT NOT NULL REFERENCES applications(id),
    environment     TEXT NOT NULL,
    label           TEXT NOT NULL,
    key_hash        TEXT NOT NULL UNIQUE,          -- sha256 of the full key
    masked          TEXT NOT NULL,
    created_by      TEXT NOT NULL,
    last_used_at    TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE developer_access (
    id             TEXT PRIMARY KEY,
    org_id         TEXT NOT NULL REFERENCES organizations(id),
    user_id        TEXT NOT NULL REFERENCES users(id),
    application_id TEXT NOT NULL REFERENCES applications(id),
    permissions    JSONB NOT NULL DEFAULT '[]',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, application_id)
);

CREATE TABLE payments (
    id                 TEXT PRIMARY KEY,
    org_id             TEXT NOT NULL REFERENCES organizations(id),
    application_id     TEXT NOT NULL REFERENCES applications(id),
    reference          TEXT NOT NULL UNIQUE,
    provider_reference TEXT,
    provider           TEXT NOT NULL,
    environment        TEXT NOT NULL,
    amount_minor       BIGINT NOT NULL,
    currency           TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'pending',
    customer           JSONB NOT NULL DEFAULT '{}',
    description        TEXT NOT NULL DEFAULT '',
    provider_response  JSONB NOT NULL DEFAULT '{}', -- redacted before storage
    fee_minor          BIGINT,
    timeline           JSONB NOT NULL DEFAULT '[]',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_payments_org_created ON payments (org_id, created_at DESC);
CREATE INDEX idx_payments_app ON payments (application_id);

CREATE TABLE webhook_endpoints (
    id             TEXT PRIMARY KEY,
    org_id         TEXT NOT NULL REFERENCES organizations(id),
    application_id TEXT NOT NULL REFERENCES applications(id),
    url            TEXT NOT NULL,
    environment    TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'active',
    events         JSONB NOT NULL DEFAULT '[]',
    secret_encrypted BYTEA,                        -- whsec_* encrypted
    secret_masked  TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE webhook_deliveries (
    id            TEXT PRIMARY KEY,
    org_id        TEXT NOT NULL REFERENCES organizations(id),
    endpoint_id   TEXT NOT NULL REFERENCES webhook_endpoints(id),
    event         TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    attempts      INT NOT NULL DEFAULT 0,
    response_code INT,
    payload       JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at  TIMESTAMPTZ
);

CREATE TABLE api_logs (
    id             TEXT PRIMARY KEY,
    org_id         TEXT NOT NULL REFERENCES organizations(id),
    application_id TEXT NOT NULL REFERENCES applications(id),
    environment    TEXT NOT NULL,
    method         TEXT NOT NULL,
    path           TEXT NOT NULL,
    status_code    INT NOT NULL,
    duration_ms    INT NOT NULL,
    source_ip      TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_api_logs_org_created ON api_logs (org_id, created_at DESC);

CREATE TABLE audit_logs (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL REFERENCES organizations(id),
    actor_name TEXT NOT NULL,
    actor_role TEXT NOT NULL,
    action     TEXT NOT NULL,
    detail     TEXT NOT NULL DEFAULT '',
    severity   TEXT NOT NULL DEFAULT 'info',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE audit_logs;
DROP TABLE api_logs;
DROP TABLE webhook_deliveries;
DROP TABLE webhook_endpoints;
DROP TABLE payments;
DROP TABLE developer_access;
DROP TABLE api_keys;
DROP TABLE applications;
DROP TABLE provider_connections;
DROP TABLE users;
DROP TABLE organizations;
