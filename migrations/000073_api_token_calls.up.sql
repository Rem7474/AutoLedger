-- The last calls an assistant made with an access: which tool, when, whether it worked and how long it took. Never the
-- arguments or the answer. The repository keeps the latest 50 per token, and a token that is deleted takes its calls.
CREATE TABLE api_token_calls (
    id BIGSERIAL PRIMARY KEY,
    token_id UUID NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
    called_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tool TEXT NOT NULL,
    vehicle_id UUID,
    outcome TEXT NOT NULL CHECK (outcome IN ('ok', 'error')),
    duration_ms INTEGER NOT NULL
);
CREATE INDEX api_token_calls_token_idx ON api_token_calls (token_id, id DESC);
