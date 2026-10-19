-- What an API token may do: 'ingest' is the behaviour every token had until now (push events, read the
-- Home Assistant views), 'read' only reads. Existing tokens keep 'ingest'; a new token states its scope.
ALTER TABLE api_tokens ADD COLUMN scope TEXT NOT NULL DEFAULT 'ingest' CHECK (scope IN ('ingest', 'read'));
ALTER TABLE api_tokens ALTER COLUMN scope DROP DEFAULT;
