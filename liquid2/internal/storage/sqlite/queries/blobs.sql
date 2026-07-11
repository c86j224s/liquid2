-- name: CreateBlob :one
INSERT INTO blobs (
  id, document_id, filename, mime_type, size, sha256, data, created_at
) VALUES (
  sqlc.arg(id), sqlc.arg(document_id), sqlc.arg(filename), sqlc.arg(mime_type),
  sqlc.arg(size), sqlc.arg(sha256), sqlc.arg(data), sqlc.arg(created_at)
) RETURNING *;

-- name: ListDocumentBlobs :many
SELECT * FROM blobs
WHERE document_id = sqlc.arg(document_id)
ORDER BY created_at ASC, id ASC;

-- name: DeleteDocumentBlobs :exec
DELETE FROM blobs WHERE document_id = sqlc.arg(document_id);
