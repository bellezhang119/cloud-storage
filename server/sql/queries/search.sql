-- name: SearchFilesAndFolders :one
SELECT
    -- Folders array
    COALESCE(
            (
                SELECT JSON_AGG(
                               JSON_BUILD_OBJECT(
                                       'id', f.id,
                                       'user_id', f.user_id,
                                       'name', f.name,
                                       'parent_id', f.parent_id,
                                       'created_at', f.created_at,
                                       'updated_at', f.updated_at
                               ) ORDER BY
                                   CASE WHEN $3 = 'name' AND $4 = 'asc' THEN f.name END ASC,
                                   CASE WHEN $3 = 'name' AND $4 = 'desc' THEN f.name END DESC,
                                   CASE WHEN $3 = 'created_at' AND $4 = 'asc' THEN f.created_at END ASC,
                                   CASE WHEN $3 = 'created_at' AND $4 = 'desc' THEN f.created_at END DESC,
                                   CASE WHEN $3 = 'updated_at' AND $4 = 'asc' THEN f.updated_at END ASC,
                                   CASE WHEN $3 = 'updated_at' AND $4 = 'desc' THEN f.updated_at END DESC
                       )
                FROM folders f
                WHERE f.user_id = $2
                  AND ($1::TEXT IS NULL OR f.name ILIKE '%' || $1 || '%')
            ),
            '[]'
    ) AS folders,

    -- Files array
    COALESCE(
            (
                SELECT JSON_AGG(
                               JSON_BUILD_OBJECT(
                                       'id', f.id,
                                       'user_id', f.user_id,
                                       'folder_id', f.folder_id,
                                       'name', f.name,
                                       'file_path', f.file_path,
                                       'size_bytes', f.size_bytes,
                                       'mime_type', f.mime_type,
                                       'created_at', f.created_at,
                                       'updated_at', f.updated_at
                               ) ORDER BY
                                   CASE WHEN $3 = 'name' AND $4 = true THEN f.name END,
                                   CASE WHEN $3 = 'name' AND $4 = false THEN f.name END DESC,
                                   CASE WHEN $3 = 'created_at' AND $4 = true THEN f.created_at END,
                                   CASE WHEN $3 = 'created_at' AND $4 = false THEN f.created_at END DESC,
                                   CASE WHEN $3 = 'updated_at' AND $4 = true THEN f.updated_at END,
                                   CASE WHEN $3 = 'updated_at' AND $4 = false THEN f.updated_at END DESC,
                                   CASE WHEN $3 = 'mime_type' AND $4 = true THEN f.mime_type END,
                                   CASE WHEN $3 = 'mime_type' AND $4 = false THEN f.mime_type END DESC,
                                   CASE WHEN $3 = 'size_bytes' AND $4 = true THEN f.size_bytes END,
                                   CASE WHEN $3 = 'size_bytes' AND $4 = false THEN f.size_bytes END DESC
                       )
                FROM files f
                WHERE f.user_id = $2
                  AND ($1::TEXT IS NULL OR f.name ILIKE '%' || $1 || '%')
                  AND ($5::TEXT[] IS NULL OR f.mime_type = ANY($5))
            ),
            '[]'
    ) AS files;
