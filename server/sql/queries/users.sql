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

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByVerificationToken :one
SELECT * FROM users WHERE verification_token = $1;

-- name: MarkUserAsVerified :exec
UPDATE users
SET is_verified = TRUE, verification_token = NULL, verification_token_expiry = NULL
WHERE id = $1;