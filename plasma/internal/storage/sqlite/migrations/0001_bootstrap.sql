CREATE TABLE IF NOT EXISTS plasma_runtime_info (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

INSERT INTO plasma_runtime_info (key, value, updated_at)
VALUES ('schema_family', 'plasma', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
