-- name: CreateJob :one
INSERT INTO jobs (
  id, kind, status, payload_json, error, attempts, created_at, updated_at,
  started_at, finished_at
) VALUES (
  sqlc.arg(id), sqlc.arg(kind), sqlc.arg(status), sqlc.arg(payload_json),
  sqlc.narg(error), sqlc.arg(attempts), sqlc.arg(created_at),
  sqlc.arg(updated_at), sqlc.narg(started_at), sqlc.narg(finished_at)
) RETURNING *;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = sqlc.arg(id);

-- name: ListJobs :many
SELECT * FROM jobs
WHERE (sqlc.narg(status) IS NULL OR status = sqlc.narg(status))
  AND (sqlc.narg(kind) IS NULL OR kind = sqlc.narg(kind))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit);

-- name: EnqueueJob :one
INSERT INTO jobs (
  id, kind, status, payload_json, error, attempts, created_at, updated_at,
  started_at, finished_at
) VALUES (
  sqlc.arg(id), sqlc.arg(kind), 'queued', sqlc.arg(payload_json),
  NULL, 0, sqlc.arg(now), sqlc.arg(now), NULL, NULL
)
ON CONFLICT(id) DO NOTHING
RETURNING *;

-- name: ClaimQueuedJob :one
UPDATE jobs
SET status = 'running',
    error = NULL,
    attempts = attempts + 1,
    updated_at = sqlc.arg(now),
    started_at = sqlc.arg(now),
    finished_at = NULL
WHERE id = (
  SELECT queued.id FROM jobs AS queued
  WHERE queued.status = 'queued'
    AND (
      queued.kind = sqlc.narg(kind_1) OR
      queued.kind = sqlc.narg(kind_2) OR
      queued.kind = sqlc.narg(kind_3) OR
      queued.kind = sqlc.narg(kind_4)
    )
  ORDER BY queued.created_at ASC, queued.id ASC
  LIMIT 1
)
RETURNING *;

-- name: UpsertJob :one
INSERT INTO jobs (
  id, kind, status, payload_json, error, attempts, created_at, updated_at,
  started_at, finished_at
) VALUES (
  sqlc.arg(id), sqlc.arg(kind), sqlc.arg(status), sqlc.arg(payload_json),
  sqlc.narg(error), sqlc.arg(attempts), sqlc.arg(created_at),
  sqlc.arg(updated_at), sqlc.narg(started_at), sqlc.narg(finished_at)
)
ON CONFLICT(id) DO UPDATE SET
  kind = excluded.kind,
  status = excluded.status,
  payload_json = excluded.payload_json,
  error = excluded.error,
  attempts = excluded.attempts,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at,
  started_at = excluded.started_at,
  finished_at = excluded.finished_at
RETURNING *;

-- name: UpdateJobState :one
UPDATE jobs
SET status = sqlc.arg(status),
    error = sqlc.narg(error),
    attempts = sqlc.arg(attempts),
    updated_at = sqlc.arg(updated_at),
    started_at = sqlc.narg(started_at),
    finished_at = sqlc.narg(finished_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: RecoverRunningJobs :exec
UPDATE jobs
SET status = sqlc.arg(status),
    error = sqlc.narg(error),
    updated_at = sqlc.arg(updated_at),
    finished_at = sqlc.narg(finished_at)
WHERE status = 'running';

-- name: SchemaVersion :one
SELECT CAST(COALESCE(MAX(version), 0) AS INTEGER) AS version FROM schema_migrations;
