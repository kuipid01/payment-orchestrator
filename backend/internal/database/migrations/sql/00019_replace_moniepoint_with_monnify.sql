-- +goose Up
-- Moniepoint connections used opaque bearer tokens and cannot be reused as
-- Monnify credentials. Keep the connection rows but require fresh verification.
CREATE TABLE monnify_legacy_provider_connections (
    connection_id TEXT PRIMARY KEY,
    old_status TEXT NOT NULL,
    old_last_verified_at TIMESTAMPTZ,
    old_updated_at TIMESTAMPTZ NOT NULL
);

INSERT INTO monnify_legacy_provider_connections (connection_id, old_status, old_last_verified_at, old_updated_at)
SELECT id, status, last_verified_at, updated_at
FROM provider_connections
WHERE provider = 'moniepoint';

UPDATE provider_connections
SET provider = 'monnify',
    status = 'error',
    last_verified_at = NULL,
    updated_at = now()
WHERE provider = 'moniepoint';

-- Payment rows retain their original provider value for historical reporting.

-- +goose Down
UPDATE provider_connections AS connection
SET provider = 'moniepoint',
    status = legacy.old_status,
    last_verified_at = legacy.old_last_verified_at,
    updated_at = legacy.old_updated_at
FROM monnify_legacy_provider_connections AS legacy
WHERE connection.id = legacy.connection_id;

DROP TABLE monnify_legacy_provider_connections;
