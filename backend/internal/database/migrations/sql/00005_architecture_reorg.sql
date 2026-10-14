-- +goose Up
-- Rename the merchant-facing identity tables to the domain vocabulary used by
-- the public API (merchant instead of organization) and add the columns the
-- payments specification requires. Data is preserved: these are renames plus
-- additive columns, never destructive rewrites.

ALTER TABLE organizations RENAME TO merchants;
ALTER TABLE merchants RENAME COLUMN name TO business_name;
ALTER TABLE merchants ADD COLUMN email TEXT NOT NULL DEFAULT '';
ALTER TABLE merchants ADD COLUMN country TEXT NOT NULL DEFAULT 'NG';
ALTER TABLE merchants ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE merchants ADD COLUMN default_currency TEXT NOT NULL DEFAULT 'NGN';
ALTER TABLE merchants ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE users RENAME COLUMN org_id TO merchant_id;

ALTER TABLE provider_connections RENAME COLUMN org_id TO merchant_id;
ALTER TABLE provider_connections ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
ALTER TABLE provider_connections ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE provider_connections ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE applications RENAME COLUMN org_id TO merchant_id;
ALTER TABLE applications ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE developer_access RENAME COLUMN org_id TO merchant_id;

-- API keys become merchant-scoped and typed (secret | public | webhook).
ALTER TABLE api_keys RENAME COLUMN org_id TO merchant_id;
ALTER TABLE api_keys RENAME COLUMN masked TO secret_masked;
ALTER TABLE api_keys ALTER COLUMN application_id DROP NOT NULL;
ALTER TABLE api_keys ADD COLUMN type TEXT NOT NULL DEFAULT 'secret';
ALTER TABLE api_keys ADD COLUMN prefix TEXT NOT NULL DEFAULT 'pc';
ALTER TABLE api_keys ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE api_keys ADD COLUMN expires_at TIMESTAMPTZ;
ALTER TABLE api_keys ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX idx_api_keys_merchant ON api_keys (merchant_id, type, environment);

ALTER TABLE payments RENAME COLUMN org_id TO merchant_id;
ALTER TABLE payments RENAME COLUMN provider_reference TO provider_payment_id;
ALTER TABLE payments ALTER COLUMN application_id DROP NOT NULL;
ALTER TABLE payments ADD COLUMN customer_id TEXT;
ALTER TABLE payments ADD COLUMN authorization_url TEXT NOT NULL DEFAULT '';
ALTER TABLE payments ADD COLUMN idempotency_key TEXT NOT NULL DEFAULT '';
ALTER TABLE payments ADD COLUMN failure_code TEXT NOT NULL DEFAULT '';
ALTER TABLE payments ADD COLUMN failure_message TEXT NOT NULL DEFAULT '';
ALTER TABLE payments ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}';
CREATE INDEX idx_payments_merchant_status ON payments (merchant_id, status, created_at DESC);
CREATE INDEX idx_payments_provider_ref ON payments (provider, provider_payment_id);
CREATE INDEX idx_payments_merchant_env ON payments (merchant_id, environment, created_at DESC);

ALTER TABLE webhook_endpoints RENAME COLUMN org_id TO merchant_id;
ALTER TABLE webhook_endpoints ADD COLUMN secret_new TEXT;
ALTER TABLE webhook_endpoints ADD COLUMN signing_secret_encrypted BYTEA;
ALTER TABLE webhook_endpoints ADD COLUMN signing_secret_masked TEXT NOT NULL DEFAULT '';
ALTER TABLE webhook_endpoints ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE webhook_endpoints ADD COLUMN failure_count INT NOT NULL DEFAULT 0;

ALTER TABLE webhook_deliveries RENAME COLUMN org_id TO merchant_id;
ALTER TABLE webhook_deliveries ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE webhook_deliveries ADD COLUMN next_attempt_at TIMESTAMPTZ;
ALTER TABLE webhook_deliveries ADD COLUMN last_error TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_webhook_deliveries_pending ON webhook_deliveries (status, next_attempt_at);

ALTER TABLE api_logs RENAME COLUMN org_id TO merchant_id;
ALTER TABLE audit_logs RENAME COLUMN org_id TO merchant_id;

-- Idempotency store: one row per (merchant, key); the request hash detects reuse
-- with a different body and the stored response makes replays safe.
CREATE TABLE idempotency_keys (
    id               TEXT PRIMARY KEY,
    merchant_id      TEXT NOT NULL REFERENCES merchants(id),
    idempotency_key  TEXT NOT NULL,
    endpoint         TEXT NOT NULL DEFAULT '',
    request_hash     TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'in_progress', -- in_progress | completed
    response_body    JSONB,
    status_code      INT,
    payment_id       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (merchant_id, idempotency_key)
);

CREATE TABLE customers (
    id          TEXT PRIMARY KEY,
    merchant_id TEXT NOT NULL REFERENCES merchants(id),
    email       TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    phone       TEXT NOT NULL DEFAULT '',
    metadata    JSONB NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_customers_merchant_email ON customers (merchant_id, lower(email)) WHERE deleted_at IS NULL;

CREATE TABLE refunds (
    id                 TEXT PRIMARY KEY,
    merchant_id        TEXT NOT NULL REFERENCES merchants(id),
    payment_id         TEXT NOT NULL REFERENCES payments(id),
    provider           TEXT NOT NULL,
    provider_refund_id TEXT,
    amount_minor       BIGINT NOT NULL,
    currency           TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'pending',
    reason             TEXT NOT NULL DEFAULT '',
    failure_message    TEXT NOT NULL DEFAULT '',
    idempotency_key    TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refunds_payment ON refunds (payment_id);
CREATE INDEX idx_refunds_merchant ON refunds (merchant_id, created_at DESC);

CREATE TABLE transactions (
    id           TEXT PRIMARY KEY,
    merchant_id  TEXT NOT NULL REFERENCES merchants(id),
    payment_id   TEXT,
    refund_id    TEXT,
    provider     TEXT NOT NULL DEFAULT '',
    environment  TEXT NOT NULL DEFAULT 'test',
    type         TEXT NOT NULL, -- payment | refund | chargeback
    direction    TEXT NOT NULL DEFAULT 'credit',
    amount_minor BIGINT NOT NULL,
    currency     TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    reference    TEXT NOT NULL DEFAULT '',
    description  TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_transactions_merchant_created ON transactions (merchant_id, created_at DESC);
CREATE INDEX idx_transactions_payment ON transactions (payment_id);

CREATE TABLE routing_rules (
    id           TEXT PRIMARY KEY,
    merchant_id  TEXT NOT NULL REFERENCES merchants(id),
    name         TEXT NOT NULL,
    currency     TEXT NOT NULL DEFAULT '',
    country      TEXT NOT NULL DEFAULT '',
    min_amount   BIGINT NOT NULL DEFAULT 0,
    max_amount   BIGINT NOT NULL DEFAULT 0,
    providers    JSONB NOT NULL DEFAULT '[]',
    priority     INT NOT NULL DEFAULT 100,
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_routing_rules_merchant ON routing_rules (merchant_id, enabled, priority);

-- Inbound provider events, deduplicated by (provider, provider_event_id).
CREATE TABLE provider_events (
    id                 TEXT PRIMARY KEY,
    merchant_id        TEXT NOT NULL REFERENCES merchants(id),
    connection_id      TEXT NOT NULL DEFAULT '',
    provider           TEXT NOT NULL,
    provider_event_id  TEXT NOT NULL,
    event_type         TEXT NOT NULL,
    payload            JSONB NOT NULL DEFAULT '{}',
    processed_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_event_id)
);

CREATE TABLE refresh_tokens (
    id          TEXT PRIMARY KEY,
    merchant_id TEXT NOT NULL REFERENCES merchants(id),
    user_id     TEXT NOT NULL REFERENCES users(id),
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id, revoked_at);

-- Backfill merchant contact details from the owning user.
UPDATE merchants m
   SET email = COALESCE((SELECT u.email FROM users u WHERE u.merchant_id = m.id ORDER BY u.created_at LIMIT 1), ''),
       updated_at = now();

-- +goose Down
DROP TABLE refresh_tokens;
DROP TABLE provider_events;
DROP TABLE routing_rules;
DROP TABLE transactions;
DROP TABLE refunds;
DROP TABLE customers;
DROP TABLE idempotency_keys;

ALTER TABLE audit_logs RENAME COLUMN merchant_id TO org_id;
ALTER TABLE api_logs RENAME COLUMN merchant_id TO org_id;
ALTER TABLE webhook_deliveries DROP COLUMN last_error, DROP COLUMN next_attempt_at, DROP COLUMN updated_at;
ALTER TABLE webhook_deliveries RENAME COLUMN merchant_id TO org_id;
ALTER TABLE webhook_endpoints DROP COLUMN failure_count, DROP COLUMN updated_at, DROP COLUMN signing_secret_masked, DROP COLUMN signing_secret_encrypted, DROP COLUMN secret_new;
ALTER TABLE webhook_endpoints RENAME COLUMN merchant_id TO org_id;
ALTER TABLE payments DROP COLUMN metadata, DROP COLUMN failure_message, DROP COLUMN failure_code, DROP COLUMN idempotency_key, DROP COLUMN authorization_url, DROP COLUMN customer_id;
ALTER TABLE payments ALTER COLUMN application_id SET NOT NULL;
ALTER TABLE payments RENAME COLUMN provider_payment_id TO provider_reference;
ALTER TABLE payments RENAME COLUMN merchant_id TO org_id;
ALTER TABLE api_keys DROP COLUMN updated_at, DROP COLUMN expires_at, DROP COLUMN name, DROP COLUMN prefix, DROP COLUMN type;
ALTER TABLE api_keys ALTER COLUMN application_id SET NOT NULL;
ALTER TABLE api_keys RENAME COLUMN secret_masked TO masked;
ALTER TABLE api_keys RENAME COLUMN merchant_id TO org_id;
ALTER TABLE developer_access RENAME COLUMN merchant_id TO org_id;
ALTER TABLE applications DROP COLUMN updated_at;
ALTER TABLE applications RENAME COLUMN merchant_id TO org_id;
ALTER TABLE provider_connections DROP COLUMN updated_at, DROP COLUMN is_default, DROP COLUMN display_name;
ALTER TABLE provider_connections RENAME COLUMN merchant_id TO org_id;
ALTER TABLE users RENAME COLUMN merchant_id TO org_id;
ALTER TABLE merchants DROP COLUMN updated_at, DROP COLUMN default_currency, DROP COLUMN status, DROP COLUMN country, DROP COLUMN email;
ALTER TABLE merchants RENAME COLUMN business_name TO name;
ALTER TABLE merchants RENAME TO organizations;
