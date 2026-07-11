-- name: CreateDocument :one
INSERT INTO documents (
  id, title, kind, folder_id, canonical_url, source_url, language, status,
  rating, created_at, updated_at, read_at, deleted_at
) VALUES (
  sqlc.arg(id), sqlc.arg(title), sqlc.arg(kind), sqlc.narg(folder_id),
  sqlc.narg(canonical_url), sqlc.narg(source_url), sqlc.narg(language),
  sqlc.arg(status), sqlc.narg(rating), sqlc.arg(created_at),
  sqlc.arg(updated_at), sqlc.narg(read_at), sqlc.narg(deleted_at)
) RETURNING *;

-- name: GetDocument :one
SELECT * FROM documents WHERE id = sqlc.arg(id);

-- name: ListDocuments :many
SELECT * FROM documents ORDER BY created_at DESC, id DESC;

-- name: UpsertDocument :one
INSERT INTO documents (
  id, title, kind, folder_id, canonical_url, source_url, language, status,
  rating, created_at, updated_at, read_at, deleted_at
) VALUES (
  sqlc.arg(id), sqlc.arg(title), sqlc.arg(kind), sqlc.narg(folder_id),
  sqlc.narg(canonical_url), sqlc.narg(source_url), sqlc.narg(language),
  sqlc.arg(status), sqlc.narg(rating), sqlc.arg(created_at),
  sqlc.arg(updated_at), sqlc.narg(read_at), sqlc.narg(deleted_at)
)
ON CONFLICT(id) DO UPDATE SET
  title = excluded.title,
  kind = excluded.kind,
  folder_id = excluded.folder_id,
  canonical_url = excluded.canonical_url,
  source_url = excluded.source_url,
  language = excluded.language,
  status = excluded.status,
  rating = excluded.rating,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at,
  read_at = excluded.read_at,
  deleted_at = excluded.deleted_at
RETURNING *;

-- name: UpdateDocumentMetadata :one
UPDATE documents
SET title = sqlc.arg(title),
    folder_id = sqlc.narg(folder_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SoftDeleteDocument :one
UPDATE documents
SET deleted_at = sqlc.arg(deleted_at),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: CreateDocumentContent :one
INSERT INTO document_contents (
  id, document_id, role, format, language, content, source_content_id, created_at
) VALUES (
  sqlc.arg(id), sqlc.arg(document_id), sqlc.arg(role), sqlc.arg(format),
  sqlc.narg(language), sqlc.arg(content), sqlc.narg(source_content_id),
  sqlc.arg(created_at)
) RETURNING *;

-- name: ListDocumentContents :many
SELECT * FROM document_contents
WHERE document_id = sqlc.arg(document_id)
ORDER BY created_at ASC, id ASC;

-- name: DeleteDocumentContents :exec
DELETE FROM document_contents WHERE document_id = sqlc.arg(document_id);
