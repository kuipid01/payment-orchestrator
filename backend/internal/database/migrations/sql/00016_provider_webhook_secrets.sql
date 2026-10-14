-- +goose Up
-- Per-connection inbound webhook configuration.
--
-- Paystack and Moniepoint sign their webhooks with the merchant's secret key, so
-- those connections need no extra value: verification falls back to the stored
-- credential. Flutterwave instead authenticates with a "verif-hash" that the
-- merchant chooses in its own dashboard, independent of the API secret key, so
-- that hash must be stored per connection for the signature check to succeed.
--
-- The secret is encrypted with the platform vault key and only its masked form
-- is ever returned by an API. webhook_last_event_at drives the "last event
-- received" health signal in the dashboard and is stamped on every verified
-- inbound webhook.
ALTER TABLE provider_connections ADD COLUMN webhook_secret_encrypted BYTEA;
ALTER TABLE provider_connections ADD COLUMN webhook_secret_masked TEXT NOT NULL DEFAULT '';
ALTER TABLE provider_connections ADD COLUMN webhook_last_event_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE provider_connections DROP COLUMN webhook_last_event_at;
ALTER TABLE provider_connections DROP COLUMN webhook_secret_masked;
ALTER TABLE provider_connections DROP COLUMN webhook_secret_encrypted;
