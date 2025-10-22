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

-- name: ListFoldersSharedWithUser :many
SELECT fo.*
FROM folder_shares fs
JOIN folders fo ON fo.id = fs.folder_id
WHERE fs.shared_user_id = $1;

-- name: GetSharedSubfolders :many
SELECT *
FROM folders
WHERE parent_id = $1;

-- name: GetFilesInSharedFolder :many
SELECT *
FROM files
WHERE folder_id = $1;

-- name: DeleteFolderShare :execrows
DELETE FROM folder_shares
WHERE folder_id = $1 AND shared_user_id = $2;

-- name: CheckUserFolderAccess :one
WITH RECURSIVE parent_folders AS (
  SELECT f.id, f.parent_id
  FROM folders f
  WHERE f.id = $1
  UNION ALL
  SELECT f2.id, f2.parent_id
  FROM folders f2
  JOIN parent_folders pf ON f2.id = pf.parent_id
)
SELECT EXISTS (
  SELECT 1
  FROM folder_shares fsh
  WHERE fsh.shared_user_id = $2
  AND fsh.folder_id IN (SELECT pf.id FROM parent_folders pf)
) AS has_access;

