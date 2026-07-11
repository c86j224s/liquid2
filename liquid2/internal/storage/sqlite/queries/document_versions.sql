-- name: CreateDocumentVersion :one
INSERT INTO document_versions (
  id, document_id, sequence, mutation_kind, title, content_snapshot_json,
  metadata_snapshot_json, created_at
) VALUES (
  sqlc.arg(id), sqlc.arg(document_id), sqlc.arg(sequence),
  sqlc.arg(mutation_kind), sqlc.arg(title), sqlc.arg(content_snapshot_json),
  sqlc.arg(metadata_snapshot_json), sqlc.arg(created_at)
) RETURNING *;

-- name: ListDocumentVersions :many
SELECT * FROM document_versions
WHERE document_id = sqlc.arg(document_id)
ORDER BY sequence ASC, id ASC;
