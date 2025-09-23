-- name: LogFileActivity :one
INSERT INTO file_activity (file_id, user_id, action, details)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListFileActivity :many
SELECT * FROM file_activity
WHERE file_id = $1
ORDER BY created_at DESC;

-- name: ListUserActivity :many
SELECT * FROM file_activity
WHERE user_id = $1
ORDER BY created_at DESC;
