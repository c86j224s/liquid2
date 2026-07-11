CREATE TABLE IF NOT EXISTS plasma_reports (
  report_id TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  object_kind TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  title TEXT NOT NULL,
  active_version_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_reports_mission_state
  ON plasma_reports (mission_id, state);

CREATE TABLE IF NOT EXISTS plasma_report_versions (
  report_version_id TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  object_kind TEXT NOT NULL,
  report_id TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  base_version_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  root_block_id TEXT NOT NULL,
  block_ids_json TEXT NOT NULL DEFAULT '[]',
  included_evidence_scope_json TEXT NOT NULL DEFAULT '{}',
  created_event_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (report_id) REFERENCES plasma_reports (report_id),
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_report_versions_report
  ON plasma_report_versions (report_id, created_at);

CREATE INDEX IF NOT EXISTS idx_plasma_report_versions_mission_state
  ON plasma_report_versions (mission_id, state);

CREATE TABLE IF NOT EXISTS plasma_report_blocks (
  block_id TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  object_kind TEXT NOT NULL,
  report_version_id TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  block_type TEXT NOT NULL,
  parent_block_id TEXT NOT NULL DEFAULT '',
  block_order INTEGER NOT NULL,
  content_json TEXT NOT NULL DEFAULT '{}',
  source_refs_json TEXT NOT NULL DEFAULT '{}',
  authorship_json TEXT NOT NULL DEFAULT '{}',
  approval_json TEXT NOT NULL DEFAULT '{}',
  FOREIGN KEY (report_version_id) REFERENCES plasma_report_versions (report_version_id),
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_report_blocks_version_order
  ON plasma_report_blocks (report_version_id, block_order);
