-- name: CreateUser :one
INSERT INTO users (
    email,
    password_hash,
    is_verified,
    verification_token,
    verification_token_expiry,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
RETURNING id, email, password_hash, is_verified, verification_token, verification_token_expiry, created_at, updated_at;
