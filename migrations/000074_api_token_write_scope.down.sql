UPDATE api_tokens SET scope = 'read' WHERE scope = 'write';
ALTER TABLE api_tokens DROP CONSTRAINT api_tokens_scope_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_scope_check CHECK (scope IN ('ingest', 'read'));
