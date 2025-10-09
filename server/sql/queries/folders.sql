-- name: CreateFolder :one
INSERT INTO folders (user_id, name, parent_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetFolderByID :one
SELECT * FROM folders WHERE id = $1;

-- name: ListFoldersByParent :many
SELECT *
FROM folders
WHERE parent_id = $1
  AND user_id = $2
ORDER BY name;

-- name: DeleteFolder :execrows
DELETE FROM folders
WHERE id = $1 AND user_id = $2;

-- name: MoveFolder :execrows
UPDATE folders
SET parent_id = $2,
    updated_at = now()
WHERE id = $1
  AND user_id = $3;

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
SET name = $2,
    updated_at = now()
WHERE id = $1 AND user_id = $3;

-- name: UpdateFolderParent :execrows
UPDATE folders
SET parent_id = $2,
    updated_at = now()
WHERE id = $1 AND user_id = $3;