-- name: CreateFolderShare :one
INSERT INTO folder_shares (folder_id, shared_user_id, permission)
VALUES ($1, $2, 'read')
RETURNING *;

-- name: GetFolderShare :one
SELECT *
FROM folder_shares
WHERE folder_id = $1 AND shared_user_id = $2;

-- name: ListFolderShares :many
SELECT *
FROM folder_shares
WHERE folder_id = $1;

-- name: DeleteFolderShare :execrows
DELETE FROM folder_shares
WHERE folder_id = $1 AND shared_user_id = $2;

-- name: CanUserAccessFolder :one
SELECT 1
FROM folders fo
LEFT JOIN folder_shares fs ON fs.folder_id = fo.id
WHERE fo.id = $1 AND (fo.user_id = $2 OR fs.shared_user_id = $2)
LIMIT 1;
