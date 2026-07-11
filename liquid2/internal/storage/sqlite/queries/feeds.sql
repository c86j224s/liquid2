-- name: CreateFeed :one
INSERT INTO feeds (
  id, url, title, folder_id, enabled, last_checked_at, created_at, updated_at
) VALUES (
  sqlc.arg(id), sqlc.arg(url), sqlc.narg(title), sqlc.narg(folder_id),
  sqlc.arg(enabled), sqlc.narg(last_checked_at), sqlc.arg(created_at),
  sqlc.arg(updated_at)
) RETURNING *;

-- name: GetFeed :one
SELECT * FROM feeds WHERE id = sqlc.arg(id);

-- name: GetFeedByURL :one
SELECT * FROM feeds WHERE url = sqlc.arg(url);

-- name: ListFeeds :many
SELECT * FROM feeds ORDER BY created_at DESC, id DESC;

-- name: UpsertFeed :one
INSERT INTO feeds (
  id, url, title, folder_id, enabled, last_checked_at, created_at, updated_at
) VALUES (
  sqlc.arg(id), sqlc.arg(url), sqlc.narg(title), sqlc.narg(folder_id),
  sqlc.arg(enabled), sqlc.narg(last_checked_at), sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
ON CONFLICT(id) DO UPDATE SET
  url = excluded.url,
  title = excluded.title,
  folder_id = excluded.folder_id,
  enabled = excluded.enabled,
  last_checked_at = excluded.last_checked_at,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at
RETURNING *;

-- name: DeleteFeed :exec
DELETE FROM feeds WHERE id = sqlc.arg(id);

-- name: CreateFeedItem :one
INSERT INTO feed_items (
  id, feed_id, document_id, guid, url, canonical_url, content_hash,
  published_at, created_at
) VALUES (
  sqlc.arg(id), sqlc.arg(feed_id), sqlc.arg(document_id), sqlc.narg(guid),
  sqlc.arg(url), sqlc.narg(canonical_url), sqlc.narg(content_hash),
  sqlc.narg(published_at), sqlc.arg(created_at)
) RETURNING *;

-- name: ListFeedItems :many
SELECT * FROM feed_items
WHERE feed_id = sqlc.arg(feed_id)
ORDER BY created_at ASC, id ASC;

-- name: GetFeedItemByDocumentID :one
SELECT * FROM feed_items
WHERE document_id = sqlc.arg(document_id)
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: UpsertFeedItem :one
INSERT INTO feed_items (
  id, feed_id, document_id, guid, url, canonical_url, content_hash,
  published_at, created_at
) VALUES (
  sqlc.arg(id), sqlc.arg(feed_id), sqlc.arg(document_id), sqlc.narg(guid),
  sqlc.arg(url), sqlc.narg(canonical_url), sqlc.narg(content_hash),
  sqlc.narg(published_at), sqlc.arg(created_at)
)
ON CONFLICT(id) DO UPDATE SET
  feed_id = excluded.feed_id,
  document_id = excluded.document_id,
  guid = excluded.guid,
  url = excluded.url,
  canonical_url = excluded.canonical_url,
  content_hash = excluded.content_hash,
  published_at = excluded.published_at,
  created_at = excluded.created_at
RETURNING *;
