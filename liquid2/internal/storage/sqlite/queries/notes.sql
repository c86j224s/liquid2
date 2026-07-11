-- name: CreateDocumentNote :one
INSERT INTO document_notes (
  id, document_id, body, format, created_at, updated_at, deleted_at
) VALUES (
  sqlc.arg(id), sqlc.arg(document_id), sqlc.arg(body), sqlc.arg(format),
  sqlc.arg(created_at), sqlc.arg(updated_at), sqlc.narg(deleted_at)
) RETURNING *;

-- name: ListDocumentNotes :many
SELECT * FROM document_notes
WHERE document_id = sqlc.arg(document_id)
  AND deleted_at IS NULL
ORDER BY created_at ASC, id ASC;

-- name: ListDocumentNotesAll :many
SELECT * FROM document_notes
WHERE document_id = sqlc.arg(document_id)
ORDER BY created_at ASC, id ASC;

-- name: GetDocumentNote :one
SELECT * FROM document_notes
WHERE document_id = sqlc.arg(document_id)
  AND id = sqlc.arg(id);

-- name: UpsertDocumentNote :one
INSERT INTO document_notes (
  id, document_id, body, format, created_at, updated_at, deleted_at
) VALUES (
  sqlc.arg(id), sqlc.arg(document_id), sqlc.arg(body), sqlc.arg(format),
  sqlc.arg(created_at), sqlc.arg(updated_at), sqlc.narg(deleted_at)
)
ON CONFLICT(id) DO UPDATE SET
  document_id = excluded.document_id,
  body = excluded.body,
  format = excluded.format,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at,
  deleted_at = excluded.deleted_at
RETURNING *;

-- name: UpdateDocumentNote :one
UPDATE document_notes
SET body = sqlc.arg(body),
    format = sqlc.arg(format),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND document_id = sqlc.arg(document_id)
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteDocumentNote :one
UPDATE document_notes
SET deleted_at = sqlc.arg(deleted_at),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND document_id = sqlc.arg(document_id)
RETURNING *;
