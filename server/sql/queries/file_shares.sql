-- name: CreateFileShare :one
INSERT INTO file_shares (file_id, shared_user_id, permission)
VALUES ($1, $2, 'read')
RETURNING *;

-- name: GetFileShare :one
SELECT *
FROM file_shares
WHERE file_id = $1 AND shared_user_id = $2;

-- name: ListFileShares :many
SELECT *
FROM file_shares
WHERE file_id = $1;

-- name: DeleteFileShare :execrows
DELETE FROM file_shares
WHERE file_id = $1 AND shared_user_id = $2;

-- name: CanUserAccessFile :one
SELECT 1
FROM files f
LEFT JOIN file_shares fs ON fs.file_id = f.id
WHERE f.id = $1 AND (f.user_id = $2 OR fs.shared_user_id = $2)
LIMIT 1;
