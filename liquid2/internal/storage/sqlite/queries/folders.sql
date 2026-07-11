-- name: CreateFolder :one
INSERT INTO folders (
  id, parent_id, name, system_role, sort_order, created_at, updated_at
) VALUES (
  sqlc.arg(id), sqlc.narg(parent_id), sqlc.arg(name),
  sqlc.narg(system_role), sqlc.arg(sort_order), sqlc.arg(created_at),
  sqlc.arg(updated_at)
) RETURNING *;

-- name: GetFolder :one
SELECT * FROM folders WHERE id = sqlc.arg(id);

-- name: ListFolders :many
SELECT * FROM folders ORDER BY parent_id, sort_order, name, id;

-- name: UpsertFolder :one
INSERT INTO folders (
  id, parent_id, name, system_role, sort_order, created_at, updated_at
) VALUES (
  sqlc.arg(id), sqlc.narg(parent_id), sqlc.arg(name),
  sqlc.narg(system_role), sqlc.arg(sort_order), sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
ON CONFLICT(id) DO UPDATE SET
  parent_id = excluded.parent_id,
  name = excluded.name,
  system_role = excluded.system_role,
  sort_order = excluded.sort_order,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at
RETURNING *;

-- name: DeleteFolder :exec
DELETE FROM folders WHERE id = sqlc.arg(id);
