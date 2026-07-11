CREATE TABLE app_sequence (
  name TEXT PRIMARY KEY,
  value INTEGER NOT NULL,
  CHECK (length(trim(name)) > 0),
  CHECK (value >= 0)
);
