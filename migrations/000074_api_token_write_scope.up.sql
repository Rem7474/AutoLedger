-- A grant given to an assistant can also write ('write' includes 'read'): it lets the MCP tools that record
-- carpools run. A token created from the account page keeps 'ingest' or 'read'.
ALTER TABLE api_tokens DROP CONSTRAINT api_tokens_scope_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_scope_check CHECK (scope IN ('ingest', 'read', 'write'));
