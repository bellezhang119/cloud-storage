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

-- name: ListFilesSharedWithUser :many
SELECT f.*
FROM file_shares fs
JOIN files f ON f.id = fs.file_id
WHERE fs.shared_user_id = $1;

-- name: DeleteFileShare :execrows
DELETE FROM file_shares
WHERE file_id = $1 AND shared_user_id = $2;

-- name: CheckUserFileAccess :one
WITH RECURSIVE parent_folders AS (
  SELECT f.id, f.parent_id
  FROM folders f
  WHERE f.id = (SELECT folder_id FROM files WHERE id = $1)
  UNION ALL
  SELECT f2.id, f2.parent_id
  FROM folders f2
  JOIN parent_folders pf ON f2.id = pf.parent_id
)
SELECT EXISTS (
  -- Case 1: Direct file share
  SELECT 1
  FROM file_shares fsh
  WHERE fsh.file_id = $1
  AND fsh.shared_user_id = $2

  UNION

  -- Case 2: Inherited access from shared folder(s)
  SELECT 1
  FROM folder_shares fldrsh
  WHERE fldrsh.shared_user_id = $2
  AND fldrsh.folder_id IN (SELECT id FROM parent_folders)
) AS has_access;
