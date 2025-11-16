-- name: CreateFolder :one
INSERT INTO folders (user_id, name, parent_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetFolderByID :one
SELECT * FROM folders WHERE id = $1 AND user_id = $2;

-- name: ListFoldersByParent :many
SELECT *
FROM folders
WHERE user_id = $1
  AND (
    (parent_id = $2 AND $2 IS NOT NULL)
    OR (parent_id IS NULL AND $2 IS NULL)
  )
ORDER BY name;

-- name: GetFolderByNameInParent :one
SELECT *
FROM folders
WHERE 
    user_id = $1 AND
    name = $2 
  AND (
    (parent_id = $3 AND $3 IS NOT NULL)
    OR (parent_id IS NULL AND $3 IS NULL)
  )
LIMIT 1;

-- name: GetFolderFullPath :one
WITH RECURSIVE folder_path AS (
    -- Anchor: start with the target folder
    SELECT 
        f.id AS folder_id,
        f.name AS folder_name,
        f.parent_id AS folder_parent_id,
        f.user_id AS folder_user_id,
        1 AS level
    FROM folders f
    WHERE f.id = $1 AND f.user_id = $2

    UNION ALL

    -- Recursive: join parent folders
    SELECT 
        f.id AS folder_id,
        f.name AS folder_name,
        f.parent_id AS folder_parent_id,
        f.user_id AS folder_user_id,
        fp.level + 1 AS level
    FROM folders f
    JOIN folder_path fp ON f.id = fp.folder_parent_id
    WHERE f.user_id = $2
)
SELECT string_agg(folder_name, '/' ORDER BY level DESC)::TEXT AS full_path
FROM folder_path;

-- name: DeleteFolders :execrows
DELETE FROM folders
WHERE id = ANY($1::uuid[]) AND user_id = $2;

-- name: ListFoldersRecursive :many
WITH RECURSIVE subfolders AS (
    SELECT f0.id, f0.user_id, f0.name, f0.parent_id, f0.created_at, f0.updated_at
    FROM folders f0
    WHERE f0.id = $1 AND f0.user_id = $2

    UNION ALL

    SELECT f.id, f.user_id, f.name, f.parent_id, f.created_at, f.updated_at
    FROM folders f
    INNER JOIN subfolders s ON f.parent_id = s.id
    WHERE f.user_id = $2
)
SELECT *
FROM subfolders;

-- name: UpdateFolderMetadata :execrows
UPDATE folders
SET name = $3,
    updated_at = now()
WHERE id = $1 AND user_id = $2;

-- name: UpdateFoldersParent :execrows
UPDATE folders
SET parent_id = $3,
    updated_at = now()
WHERE id = ANY($1::uuid[]) AND user_id = $2;
