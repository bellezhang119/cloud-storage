-- +goose Up

CREATE TABLE refresh_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    revoked BOOL NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- +goose Down

DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP TABLE refresh_tokens;