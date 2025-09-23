-- name: CreateFile :one
INSERT INTO files (folder_id, user_id, name, file_path, size_bytes, mime_type)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetFileByID :one
SELECT * FROM files WHERE id = $1;

-- name: GetFileByNameInFolder :one
SELECT * FROM files
WHERE folder_id = $1 AND name = $2 AND is_trashed = FALSE;

-- name: ListFilesInFolder :many
SELECT * FROM files
WHERE folder_id = $1 AND is_trashed = FALSE
ORDER BY name;

-- name: TrashFile :exec
UPDATE files
SET is_trashed = TRUE, deleted_at = now()
WHERE id = $1 AND user_id = $2;

-- name: RestoreFile :exec
UPDATE files
SET is_trashed = FALSE, deleted_at = NULL
WHERE id = $1 AND user_id = $2;

-- name: PermanentlyDeleteFile :exec
DELETE FROM files WHERE id = $1 AND user_id = $2;

-- name: UpdateFileMetadata :exec
UPDATE files
SET name = $2, updated_at = now()
WHERE id = $1 AND user_id = $3;
