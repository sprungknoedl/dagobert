-- The 'Dagobert' key type labeled the worker process authenticating back over
-- the HTTP API. The worker now runs in-process against the store directly, so
-- this key type is obsolete. Reassign any existing keys to the generic 'API'
-- type before removing the enum value.
UPDATE keys SET type = 'API' WHERE type = 'Dagobert';
DELETE FROM enums WHERE category = 'KeyTypes' AND name = 'Dagobert';
