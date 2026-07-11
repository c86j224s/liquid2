CREATE TABLE IF NOT EXISTS plasma_raw_artifacts (
  artifact_id TEXT PRIMARY KEY,
  mission_id TEXT NOT NULL,
  media_type TEXT NOT NULL,
  byte_size INTEGER NOT NULL,
  sha256 TEXT NOT NULL,
  storage_uri TEXT NOT NULL,
  filename TEXT NOT NULL DEFAULT '',
  producer_type TEXT NOT NULL,
  producer_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  content_blob BLOB NOT NULL,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_raw_artifacts_mission
  ON plasma_raw_artifacts (mission_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_plasma_raw_artifacts_mission_sha
  ON plasma_raw_artifacts (mission_id, sha256);

CREATE TABLE IF NOT EXISTS plasma_source_snapshots (
  snapshot_id TEXT PRIMARY KEY,
  mission_id TEXT NOT NULL,
  connector_id TEXT NOT NULL,
  connector_type TEXT NOT NULL,
  external_source_id TEXT NOT NULL DEFAULT '',
  external_uri TEXT NOT NULL DEFAULT '',
  external_version TEXT NOT NULL DEFAULT '',
  connector_version TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  captured_at TEXT NOT NULL,
  external_updated_at TEXT NOT NULL DEFAULT '',
  content_hash_algorithm TEXT NOT NULL,
  content_hash_value TEXT NOT NULL,
  locators_json TEXT NOT NULL DEFAULT '[]',
  access_visibility TEXT NOT NULL DEFAULT 'private',
  access_license TEXT NOT NULL DEFAULT 'unknown',
  retrieval_policy TEXT NOT NULL DEFAULT 'snapshot_only',
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_source_snapshots_mission
  ON plasma_source_snapshots (mission_id);

CREATE TABLE IF NOT EXISTS plasma_source_snapshot_artifacts (
  snapshot_id TEXT NOT NULL,
  artifact_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  PRIMARY KEY (snapshot_id, artifact_id),
  FOREIGN KEY (snapshot_id) REFERENCES plasma_source_snapshots (snapshot_id),
  FOREIGN KEY (artifact_id) REFERENCES plasma_raw_artifacts (artifact_id)
);
