-- +goose Up
-- The endpoint's own reply, kept so a merchant can see why their endpoint
-- rejected a webhook instead of only the status code it returned.
--
-- The dispatcher stores at most the first 2 KB of the body: enough for a
-- human-readable error message, bounded so one misbehaving or hostile endpoint
-- cannot fill the table. An empty string means no body was read (a failure that
-- never reached an endpoint, or a body that was empty).
ALTER TABLE webhook_deliveries ADD COLUMN response_body TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE webhook_deliveries DROP COLUMN response_body;
