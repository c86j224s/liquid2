-- name: NextSequence :one
INSERT INTO app_sequence (name, value)
VALUES (sqlc.arg(name), 1)
ON CONFLICT(name) DO UPDATE SET value = value + 1
RETURNING value;
