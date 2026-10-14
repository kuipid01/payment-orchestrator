-- +goose Up
-- Subscriptions: what plan an organization is on, and every change to it.
--
-- Two deliberate omissions, both recorded here so nobody "fixes" them later:
--
-- 1. merchants gains no plan column. Entitlements resolve through
--    merchant_subscriptions (cached briefly) and nowhere else. Denormalising the
--    plan onto the merchant is how the pricing page and the billing page ended
--    up claiming different free-tier limits for the same product.
--
-- 2. There is no catalog table. Prices live in internal/billing/catalog.go, so a
--    price change is a deploy and every reader — the marketing page, the
--    subscription page, and the enforcement gates — reads one value instead of
--    three that can drift.
--
-- Usage is not stored either: applications, seats, payments and routed volume
-- are measured from the tables that own those facts. A counter that can disagree
-- with the payments table is a support ticket we would deserve.

CREATE TABLE merchant_subscriptions (
    id                       TEXT PRIMARY KEY,
    merchant_id              TEXT NOT NULL UNIQUE REFERENCES merchants(id),
    plan                     TEXT NOT NULL DEFAULT 'free',   -- free | lite | pro | enterprise
    status                   TEXT NOT NULL DEFAULT 'free',   -- free | trialing | active | past_due | pending_payment | cancelled
    interval                 TEXT NOT NULL DEFAULT 'none',   -- monthly | quarterly | annual | none
    -- The band and currency a plan was sold in. Recorded rather than re-derived
    -- on read: a merchant who moves country is not repriced mid-term.
    band                     TEXT NOT NULL DEFAULT '',
    currency                 TEXT NOT NULL DEFAULT '',
    provider                 TEXT NOT NULL DEFAULT '',       -- paystack | stripe
    provider_subscription_id TEXT NOT NULL DEFAULT '',
    current_period_start     TIMESTAMPTZ,
    current_period_end       TIMESTAMPTZ,
    cancel_at_period_end     BOOLEAN NOT NULL DEFAULT FALSE,
    trial_ends_at            TIMESTAMPTZ,
    -- trial_used is permanent, including after a cancellation and after a
    -- return to Free: the trial is once per organization.
    trial_used               BOOLEAN NOT NULL DEFAULT FALSE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_merchant_subscriptions_status ON merchant_subscriptions (status, current_period_end);

-- Append-only trail of subscription state changes. Every plan change is a
-- billing fact, so it is recorded here and mirrored into audit_logs.
CREATE TABLE subscription_events (
    id         TEXT PRIMARY KEY,
    merchant_id TEXT NOT NULL REFERENCES merchants(id),
    type       TEXT NOT NULL,               -- subscribe | change_plan | trial_started | trial_ended | cancel | resume | checkout_requested | payment_failed | reconciled
    from_plan  TEXT NOT NULL DEFAULT '',
    to_plan    TEXT NOT NULL DEFAULT '',
    source     TEXT NOT NULL DEFAULT 'api', -- api | provider_webhook | admin | system
    payload    JSONB NOT NULL DEFAULT '{}',
    actor_name TEXT NOT NULL DEFAULT '',
    actor_role TEXT NOT NULL DEFAULT '',
    detail     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_subscription_events_merchant ON subscription_events (merchant_id, created_at DESC);

-- +goose Down
DROP TABLE subscription_events;
DROP TABLE merchant_subscriptions;
