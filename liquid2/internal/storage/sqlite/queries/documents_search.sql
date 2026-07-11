-- name: ListDocumentIDsRecent :many
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
),
cursor_doc AS (
  SELECT * FROM documents WHERE id = sqlc.narg(cursor_id)
)
SELECT documents.id
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
  )
  AND (
    sqlc.narg(cursor_id) IS NULL OR EXISTS (
      SELECT 1 FROM cursor_doc
      WHERE documents.updated_at < cursor_doc.updated_at
        OR (
          documents.updated_at = cursor_doc.updated_at
          AND documents.created_at < cursor_doc.created_at
        )
        OR (
          documents.updated_at = cursor_doc.updated_at
          AND documents.created_at = cursor_doc.created_at
          AND documents.id < cursor_doc.id
        )
    )
  )
ORDER BY documents.updated_at DESC, documents.created_at DESC, documents.id DESC
LIMIT sqlc.arg(limit_rows);

-- name: ListDocumentIDsCreatedDesc :many
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
),
cursor_doc AS (
  SELECT * FROM documents WHERE id = sqlc.narg(cursor_id)
)
SELECT documents.id
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
  )
  AND (
    sqlc.narg(cursor_id) IS NULL OR EXISTS (
      SELECT 1 FROM cursor_doc
      WHERE documents.created_at < cursor_doc.created_at
        OR (
          documents.created_at = cursor_doc.created_at
          AND documents.id < cursor_doc.id
        )
    )
  )
ORDER BY documents.created_at DESC, documents.id DESC
LIMIT sqlc.arg(limit_rows);

-- name: ListDocumentIDsRatingDesc :many
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
),
cursor_doc AS (
  SELECT * FROM documents WHERE id = sqlc.narg(cursor_id)
)
SELECT documents.id
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
  )
  AND (
    sqlc.narg(cursor_id) IS NULL OR EXISTS (
      SELECT 1 FROM cursor_doc
      WHERE COALESCE(documents.rating, 0) < COALESCE(cursor_doc.rating, 0)
        OR (
          COALESCE(documents.rating, 0) = COALESCE(cursor_doc.rating, 0)
          AND documents.updated_at < cursor_doc.updated_at
        )
        OR (
          COALESCE(documents.rating, 0) = COALESCE(cursor_doc.rating, 0)
          AND documents.updated_at = cursor_doc.updated_at
          AND documents.created_at < cursor_doc.created_at
        )
        OR (
          COALESCE(documents.rating, 0) = COALESCE(cursor_doc.rating, 0)
          AND documents.updated_at = cursor_doc.updated_at
          AND documents.created_at = cursor_doc.created_at
          AND documents.id < cursor_doc.id
        )
    )
  )
ORDER BY COALESCE(documents.rating, 0) DESC, documents.updated_at DESC,
  documents.created_at DESC, documents.id DESC
LIMIT sqlc.arg(limit_rows);

-- name: SearchDocumentIDsRelevance :many
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
),
cursor_doc AS (
  SELECT cursor_documents.*, bm25(documents_fts) AS cursor_rank
  FROM documents_fts(sqlc.arg(query)) AS cursor_matches
  JOIN documents AS cursor_documents ON cursor_documents.id = cursor_matches.document_id
  WHERE cursor_documents.id = sqlc.narg(cursor_id)
)
SELECT documents.id
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
  )
  AND (
    sqlc.narg(cursor_id) IS NULL OR EXISTS (
      SELECT 1 FROM cursor_doc
      WHERE bm25(documents_fts) > cursor_doc.cursor_rank
        OR (
          bm25(documents_fts) = cursor_doc.cursor_rank
          AND documents.updated_at < cursor_doc.updated_at
        )
        OR (
          bm25(documents_fts) = cursor_doc.cursor_rank
          AND documents.updated_at = cursor_doc.updated_at
          AND documents.created_at < cursor_doc.created_at
        )
        OR (
          bm25(documents_fts) = cursor_doc.cursor_rank
          AND documents.updated_at = cursor_doc.updated_at
          AND documents.created_at = cursor_doc.created_at
          AND documents.id < cursor_doc.id
        )
    )
  )
ORDER BY bm25(documents_fts), documents.updated_at DESC, documents.created_at DESC,
  documents.id DESC
LIMIT sqlc.arg(limit_rows);

-- name: SearchDocumentIDsRecent :many
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
),
cursor_doc AS (
  SELECT * FROM documents WHERE id = sqlc.narg(cursor_id)
)
SELECT documents.id
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
  )
  AND (
    sqlc.narg(cursor_id) IS NULL OR EXISTS (
      SELECT 1 FROM cursor_doc
      WHERE documents.updated_at < cursor_doc.updated_at
        OR (
          documents.updated_at = cursor_doc.updated_at
          AND documents.created_at < cursor_doc.created_at
        )
        OR (
          documents.updated_at = cursor_doc.updated_at
          AND documents.created_at = cursor_doc.created_at
          AND documents.id < cursor_doc.id
        )
    )
  )
ORDER BY documents.updated_at DESC, documents.created_at DESC, documents.id DESC
LIMIT sqlc.arg(limit_rows);

-- name: SearchDocumentIDsCreatedDesc :many
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
),
cursor_doc AS (
  SELECT * FROM documents WHERE id = sqlc.narg(cursor_id)
)
SELECT documents.id
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
  )
  AND (
    sqlc.narg(cursor_id) IS NULL OR EXISTS (
      SELECT 1 FROM cursor_doc
      WHERE documents.created_at < cursor_doc.created_at
        OR (
          documents.created_at = cursor_doc.created_at
          AND documents.id < cursor_doc.id
        )
    )
  )
ORDER BY documents.created_at DESC, documents.id DESC
LIMIT sqlc.arg(limit_rows);

-- name: SearchDocumentIDsRatingDesc :many
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
),
cursor_doc AS (
  SELECT * FROM documents WHERE id = sqlc.narg(cursor_id)
)
SELECT documents.id
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
  )
  AND (
    sqlc.narg(cursor_id) IS NULL OR EXISTS (
      SELECT 1 FROM cursor_doc
      WHERE COALESCE(documents.rating, 0) < COALESCE(cursor_doc.rating, 0)
        OR (
          COALESCE(documents.rating, 0) = COALESCE(cursor_doc.rating, 0)
          AND documents.updated_at < cursor_doc.updated_at
        )
        OR (
          COALESCE(documents.rating, 0) = COALESCE(cursor_doc.rating, 0)
          AND documents.updated_at = cursor_doc.updated_at
          AND documents.created_at < cursor_doc.created_at
        )
        OR (
          COALESCE(documents.rating, 0) = COALESCE(cursor_doc.rating, 0)
          AND documents.updated_at = cursor_doc.updated_at
          AND documents.created_at = cursor_doc.created_at
          AND documents.id < cursor_doc.id
        )
    )
  )
ORDER BY COALESCE(documents.rating, 0) DESC, documents.updated_at DESC,
  documents.created_at DESC, documents.id DESC
LIMIT sqlc.arg(limit_rows);
