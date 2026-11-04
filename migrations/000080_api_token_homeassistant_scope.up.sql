-- 'homeassistant' is both directions of the Home Assistant integration: it pushes charging events (like 'ingest')
-- and reads the integration views and the read API (like 'read'). A token can be created with it from the account
-- page, and an OAuth grant can hold it.
ALTER TABLE api_tokens DROP CONSTRAINT api_tokens_scope_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_scope_check CHECK (scope IN ('ingest', 'read', 'write', 'homeassistant'));
