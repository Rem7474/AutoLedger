UPDATE api_tokens SET scope = 'ingest' WHERE scope = 'homeassistant';
ALTER TABLE api_tokens DROP CONSTRAINT api_tokens_scope_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_scope_check CHECK (scope IN ('ingest', 'read', 'write'));
