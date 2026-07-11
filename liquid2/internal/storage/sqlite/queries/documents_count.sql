-- name: CountDocumentIDs :one
WITH RECURSIVE folder_scope(id) AS (
  SELECT sqlc.narg(folder_id) WHERE sqlc.narg(folder_id) IS NOT NULL
  UNION ALL
  SELECT folders.id FROM folders
  JOIN folder_scope ON folders.parent_id = folder_scope.id
  WHERE sqlc.arg(include_folder_descendants) != 0
),
trash_scope(id) AS (
  SELECT folders.id FROM folders WHERE folders.system_role = 'trash'
  UNION ALL
  SELECT folders.id FROM folders
  JOIN trash_scope ON folders.parent_id = trash_scope.id
)
SELECT COUNT(*)
FROM documents
WHERE (sqlc.arg(include_deleted) != 0 OR documents.deleted_at IS NULL)
  AND (
    sqlc.arg(include_trash) != 0 OR sqlc.narg(folder_id) IS NOT NULL OR NOT EXISTS (
      SELECT 1 FROM trash_scope
      WHERE trash_scope.id = documents.folder_id
    )
  )
  AND (sqlc.narg(status) IS NULL OR documents.status = sqlc.narg(status))
  AND (sqlc.narg(kind) IS NULL OR documents.kind = sqlc.narg(kind))
  AND (sqlc.narg(folder_id) IS NULL OR documents.folder_id IN (SELECT id FROM folder_scope))
  AND (sqlc.arg(rating_min) <= 0 OR documents.rating >= sqlc.arg(rating_min))
  AND (
    sqlc.narg(tag) IS NULL OR EXISTS (
      SELECT 1 FROM document_tags
      JOIN tags ON tags.id = document_tags.tag_id
      WHERE document_tags.document_id = documents.id
        AND tags.slug = sqlc.narg(tag)
    )
  );

-- name: CountSearchDocumentIDs :one
WITH RECURSIVE folder_scope(id) AS (
  SELECT sqlc.narg(folder_id) WHERE sqlc.narg(folder_id) IS NOT NULL
  UNION ALL
  SELECT folders.id FROM folders
  JOIN folder_scope ON folders.parent_id = folder_scope.id
  WHERE sqlc.arg(include_folder_descendants) != 0
),
trash_scope(id) AS (
  SELECT folders.id FROM folders WHERE folders.system_role = 'trash'
  UNION ALL
  SELECT folders.id FROM folders
  JOIN trash_scope ON folders.parent_id = trash_scope.id
)
SELECT COUNT(*)
FROM documents_fts(sqlc.arg(query)) AS matches
JOIN documents ON documents.id = matches.document_id
WHERE (sqlc.arg(include_deleted) != 0 OR documents.deleted_at IS NULL)
  AND (
    sqlc.arg(include_trash) != 0 OR sqlc.narg(folder_id) IS NOT NULL OR NOT EXISTS (
      SELECT 1 FROM trash_scope
      WHERE trash_scope.id = documents.folder_id
    )
  )
  AND (sqlc.narg(status) IS NULL OR documents.status = sqlc.narg(status))
  AND (sqlc.narg(kind) IS NULL OR documents.kind = sqlc.narg(kind))
  AND (sqlc.narg(folder_id) IS NULL OR documents.folder_id IN (SELECT id FROM folder_scope))
  AND (sqlc.arg(rating_min) <= 0 OR documents.rating >= sqlc.arg(rating_min))
  AND (
    sqlc.narg(tag) IS NULL OR EXISTS (
      SELECT 1 FROM document_tags
      JOIN tags ON tags.id = document_tags.tag_id
      WHERE document_tags.document_id = documents.id
        AND tags.slug = sqlc.narg(tag)
    )
  );
