-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3);

-- name: GetRefreshToken :one
SELECT token_hash, user_id, expires_at, revoked FROM refresh_tokens WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked = TRUE WHERE token_hash = $1;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens WHERE expires_at < NOW();