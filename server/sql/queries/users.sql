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
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByVerificationToken :one
SELECT * FROM users WHERE verification_token = $1;

-- name: MarkUserAsVerified :execrows
UPDATE users
SET is_verified = TRUE,
    verification_token = NULL,
    verification_token_expiry = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateVerificationToken :execrows
UPDATE users
SET verification_token = $1,
    verification_token_expiry = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE email = $3;

-- name: UpdateUserPassword :execrows
UPDATE users
SET password_hash = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateUsedStorage :execrows
UPDATE users
SET used_storage = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE id = $1;
