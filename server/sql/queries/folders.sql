-- name: CreateFolder :one
INSERT INTO folders (user_id, name, parent_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetFolderByID :one
SELECT * FROM folders WHERE id = $1;

-- name: ListFoldersByParent :many
SELECT * FROM folders
WHERE user_id = $1 AND parent_id = $2
ORDER BY name;

-- name: DeleteFolder :execrows
DELETE FROM folders
WHERE id = $1 AND user_id = $2;
