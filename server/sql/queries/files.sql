-- name: CreateFile :one
INSERT INTO files (folder_id, user_id, name, file_path, size_bytes, mime_type)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetFileByID :one
SELECT * FROM files WHERE id = $1;

-- name: GetFileByNameInFolder :one
SELECT * FROM files
WHERE folder_id = $1 AND name = $2;

-- name: ListFilesInFolder :many
SELECT *
FROM files
WHERE (folder_id = $2 OR ($2 IS NULL AND folder_id IS NULL)) AND user_id = $1
ORDER BY name;

-- name: ListFilesRecursive :many
WITH RECURSIVE subfolders AS (
    SELECT folders.id AS sf_folder_id
    FROM folders
    WHERE folders.id = $2 AND folders.user_id = $1

    UNION ALL

    SELECT f.id AS sf_folder_id
    FROM folders f
    INNER JOIN subfolders s ON f.parent_id = s.sf_folder_id
    WHERE f.user_id = $1
)
SELECT 
    f.id AS file_id,
    f.folder_id AS folder_id,
    f.user_id AS user_id,
    f.name AS name,
    f.file_path AS file_path,
    f.size_bytes AS size_bytes,
    f.mime_type AS mime_type,
    f.created_at AS created_at,
    f.updated_at AS updated_at
FROM files f
INNER JOIN subfolders sf ON f.folder_id = sf.sf_folder_id
WHERE f.user_id = $1
ORDER BY f.name;

-- name: DeleteFile :execrows
DELETE FROM files
WHERE id = $1 AND user_id = $2;

-- name: UpdateFileMetadata :execrows
UPDATE files
SET
    name       = COALESCE($3::text, name),
    file_path  = COALESCE($4::text, file_path),
    size_bytes = COALESCE($5::bigint, size_bytes),
    mime_type  = COALESCE($6::text, mime_type),
    updated_at = now()
WHERE id = $1
  AND user_id = $2;