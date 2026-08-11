CREATE TABLE IF NOT EXISTS plasma_report_runs (
  run_id TEXT PRIMARY KEY,
  mission_id TEXT NOT NULL,
  root_pending_event_id TEXT NOT NULL,
  lifecycle_state TEXT NOT NULL CHECK (lifecycle_state IN ('active', 'completed', 'failed', 'canceled', 'ambiguous', 'purged')),
  revision INTEGER NOT NULL CHECK (revision >= 1),
  title TEXT NOT NULL DEFAULT '',
  final_artifact_id TEXT NOT NULL DEFAULT '',
  registration_status TEXT NOT NULL CHECK (registration_status IN ('native', 'backfilled')),
  purged_at TEXT NOT NULL DEFAULT '',
  purged_by_type TEXT NOT NULL DEFAULT '',
  purged_by_id TEXT NOT NULL DEFAULT '',
  usage_record_count INTEGER NOT NULL DEFAULT 0,
  usage_available_count INTEGER NOT NULL DEFAULT 0,
  usage_unavailable_count INTEGER NOT NULL DEFAULT 0,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  cached_input_tokens INTEGER NOT NULL DEFAULT 0,
  uncached_input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  reasoning_output_tokens INTEGER NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  usage_partial INTEGER NOT NULL DEFAULT 0 CHECK (usage_partial IN (0, 1)),
  aggregation_version TEXT NOT NULL DEFAULT 'report_usage.v1',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_report_runs_mission
  ON plasma_report_runs (mission_id, lifecycle_state, updated_at);

CREATE TABLE IF NOT EXISTS plasma_report_run_events (
  run_id TEXT NOT NULL,
  event_id TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  event_role TEXT NOT NULL,
  attempt_event_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  PRIMARY KEY (run_id, event_id),
  FOREIGN KEY (run_id) REFERENCES plasma_report_runs (run_id) ON DELETE CASCADE,
  FOREIGN KEY (event_id) REFERENCES plasma_ledger_events (event_id) ON DELETE CASCADE,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_report_run_events_event
  ON plasma_report_run_events (event_id);

CREATE INDEX IF NOT EXISTS idx_plasma_report_run_events_mission
  ON plasma_report_run_events (mission_id, run_id);

CREATE TABLE IF NOT EXISTS plasma_report_run_artifacts (
  run_id TEXT NOT NULL,
  artifact_id TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  artifact_role TEXT NOT NULL CHECK (artifact_role IN ('input', 'intermediate', 'final', 'derivative')),
  ownership TEXT NOT NULL CHECK (ownership IN ('created', 'referenced')),
  attempt_event_id TEXT NOT NULL DEFAULT '',
  source_event_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  PRIMARY KEY (run_id, artifact_id),
  FOREIGN KEY (run_id) REFERENCES plasma_report_runs (run_id) ON DELETE CASCADE,
  FOREIGN KEY (artifact_id) REFERENCES plasma_raw_artifacts (artifact_id) ON DELETE CASCADE,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_report_run_artifacts_artifact
  ON plasma_report_run_artifacts (artifact_id);

CREATE INDEX IF NOT EXISTS idx_plasma_report_run_artifacts_mission
  ON plasma_report_run_artifacts (mission_id, run_id);
