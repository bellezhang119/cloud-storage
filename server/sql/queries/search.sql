-- name: SearchFilesAndFolders :many
WITH RECURSIVE folder_hierarchy AS (
    SELECT f.id AS folder_id, f.name AS folder_name, f.parent_id, f.user_id, f.created_at, f.updated_at
    FROM folders f
    WHERE f.id = $1
    UNION ALL
    SELECT f2.id AS folder_id, f2.name AS folder_name, f2.parent_id, f2.user_id, f2.created_at, f2.updated_at
    FROM folders f2
    INNER JOIN folder_hierarchy fh ON f2.parent_id = fh.folder_id
)
SELECT
    files.id AS id,
    files.name AS name,
    files.file_path AS file_path,
    files.size_bytes AS size_bytes,
    files.mime_type AS mime_type,
    files.created_at AS created_at,
    files.updated_at AS updated_at
FROM files
WHERE files.folder_id IN (SELECT folder_id FROM folder_hierarchy)
  AND files.user_id = $2
  AND ($3::TEXT IS NULL OR files.name ILIKE '%' || $3 || '%')
  AND ($6::TEXT[] IS NULL OR files.mime_type = ANY($6))
UNION ALL
SELECT
    fh.folder_id AS id,
    fh.folder_name AS name,
    NULL AS file_path,
    NULL AS size_bytes,
    'folder' AS mime_type,
    fh.created_at,
    fh.updated_at
FROM folder_hierarchy fh
WHERE fh.user_id = $2
  AND ($3::TEXT IS NULL OR fh.folder_name ILIKE '%' || $3 || '%')
  AND ($6::TEXT IS NULL OR 'folder' = $6)
ORDER BY
    CASE WHEN $4 = 'name' AND $5 = 'asc' THEN name END ASC,
    CASE WHEN $4 = 'name' AND $5 = 'desc' THEN name END DESC,
    CASE WHEN $4 = 'created_at' AND $5 = 'asc' THEN created_at END ASC,
    CASE WHEN $4 = 'created_at' AND $5 = 'desc' THEN created_at END DESC,
    CASE WHEN $4 = 'updated_at' AND $5 = 'asc' THEN updated_at END ASC,
    CASE WHEN $4 = 'updated_at' AND $5 = 'desc' THEN updated_at END DESC,
    CASE WHEN $4 = 'size_bytes' AND $5 = 'asc' THEN size_bytes END ASC,
    CASE WHEN $4 = 'size_bytes' AND $5 = 'desc' THEN size_bytes END DESC,
    CASE WHEN $4 = 'mime_type' AND $5 = 'asc' THEN mime_type END ASC,
    CASE WHEN $4 = 'mime_type' AND $5 = 'desc' THEN mime_type END DESC;

