-- name: ShareFile :one
INSERT INTO file_shares (file_id, shared_with, permission)
VALUES ($1, $2, $3)
ON CONFLICT (file_id, shared_with) DO UPDATE
SET permission = EXCLUDED.permission
RETURNING *;

-- name: GetFileShares :many
SELECT * FROM file_shares WHERE file_id = $1;

-- name: RemoveFileShare :execrows
DELETE FROM file_shares
WHERE file_id = $1 AND shared_with = $2;
