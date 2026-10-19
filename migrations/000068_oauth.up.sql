-- OAuth 2.1 authorization server for assistants (MCP clients): public clients that register themselves, one-minute
-- authorization codes, and grants. A grant is an api_tokens row with a short-lived access token and a rotating
-- refresh token, so it is listed and revoked from the account page like any other token.
CREATE TABLE oauth_clients (
    id TEXT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    redirect_uris TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE oauth_codes (
    code_hash VARCHAR(64) PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    code_challenge VARCHAR(128) NOT NULL,
    scope TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE api_tokens
    ADD COLUMN oauth_client_id TEXT REFERENCES oauth_clients(id) ON DELETE CASCADE,
    ADD COLUMN refresh_hash VARCHAR(64) UNIQUE,
    ADD COLUMN prev_refresh_hash VARCHAR(64),
    ADD COLUMN refresh_expires_at TIMESTAMPTZ,
    ADD CONSTRAINT api_tokens_oauth_refresh CHECK (oauth_client_id IS NULL OR (refresh_hash IS NOT NULL AND refresh_expires_at IS NOT NULL));

CREATE INDEX idx_api_tokens_oauth_client ON api_tokens(oauth_client_id) WHERE oauth_client_id IS NOT NULL;
CREATE INDEX idx_api_tokens_prev_refresh ON api_tokens(prev_refresh_hash) WHERE prev_refresh_hash IS NOT NULL;
