DELETE FROM api_tokens WHERE oauth_client_id IS NOT NULL;
ALTER TABLE api_tokens
    DROP CONSTRAINT api_tokens_oauth_refresh,
    DROP COLUMN refresh_expires_at,
    DROP COLUMN prev_refresh_hash,
    DROP COLUMN refresh_hash,
    DROP COLUMN oauth_client_id;
DROP TABLE oauth_codes;
DROP TABLE oauth_clients;
