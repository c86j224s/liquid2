-- name: GetAppSettings :one
SELECT * FROM app_settings WHERE id = 1;

-- name: UpsertAppSettings :one
INSERT INTO app_settings (
  id, feed_scheduler_enabled, feed_poll_interval_seconds, feed_next_poll_at, updated_at
) VALUES (
  1, sqlc.arg(feed_scheduler_enabled),
  sqlc.arg(feed_poll_interval_seconds), sqlc.narg(feed_next_poll_at),
  sqlc.arg(updated_at)
)
ON CONFLICT(id) DO UPDATE SET
  feed_scheduler_enabled = excluded.feed_scheduler_enabled,
  feed_poll_interval_seconds = excluded.feed_poll_interval_seconds,
  feed_next_poll_at = excluded.feed_next_poll_at,
  updated_at = excluded.updated_at
RETURNING *;
