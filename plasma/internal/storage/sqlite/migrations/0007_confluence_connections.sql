CREATE TABLE IF NOT EXISTS plasma_confluence_connections (
  connection_id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  auth_type TEXT NOT NULL,
  account_id TEXT NOT NULL DEFAULT '',
  account_name TEXT NOT NULL DEFAULT '',
  access_token TEXT NOT NULL DEFAULT '',
  refresh_token TEXT NOT NULL DEFAULT '',
  token_expires_at TEXT NOT NULL DEFAULT '',
  scopes_json TEXT NOT NULL DEFAULT '[]',
  sites_json TEXT NOT NULL DEFAULT '[]',
  revoked INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plasma_confluence_connections_updated
  ON plasma_confluence_connections (updated_at DESC, connection_id);
