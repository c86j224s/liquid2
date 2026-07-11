-- name: CreateTag :one
INSERT INTO tags (id, name, slug, created_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(created_at))
RETURNING *;

-- name: AssignDocumentTag :exec
INSERT INTO document_tags (document_id, tag_id)
VALUES (sqlc.arg(document_id), sqlc.arg(tag_id));

-- name: DeleteDocumentTags :exec
DELETE FROM document_tags WHERE document_id = sqlc.arg(document_id);

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = sqlc.arg(id);

-- name: GetTag :one
SELECT * FROM tags WHERE id = sqlc.arg(id);

-- name: GetTagBySlug :one
SELECT * FROM tags WHERE slug = sqlc.arg(slug);

-- name: TagHasDocuments :one
SELECT 1 FROM document_tags
WHERE tag_id = sqlc.arg(tag_id)
LIMIT 1;

-- name: ListTags :many
SELECT * FROM tags ORDER BY slug ASC, id ASC;

-- name: UpsertTag :one
INSERT INTO tags (id, name, slug, created_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(created_at))
ON CONFLICT(id) DO UPDATE SET
  name = excluded.name,
  slug = excluded.slug,
  created_at = excluded.created_at
RETURNING *;

-- name: ListDocumentTags :many
SELECT tags.* FROM tags
JOIN document_tags ON document_tags.tag_id = tags.id
WHERE document_tags.document_id = sqlc.arg(document_id)
ORDER BY tags.slug ASC;
