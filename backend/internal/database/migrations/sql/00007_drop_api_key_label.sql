-- +goose Up
-- 00005 added api_keys.name as the canonical key label but left the original
-- NOT NULL `label` column behind. The domain model no longer writes it, so every
-- key insert failed on the not-null constraint. Backfill, then drop it.

UPDATE api_keys SET name = label WHERE name = '' AND label <> '';
ALTER TABLE api_keys DROP COLUMN label;

-- +goose Down
ALTER TABLE api_keys ADD COLUMN label TEXT NOT NULL DEFAULT '';
UPDATE api_keys SET label = name WHERE label = '';
