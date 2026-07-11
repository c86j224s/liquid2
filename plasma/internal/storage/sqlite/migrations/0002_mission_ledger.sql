CREATE TABLE IF NOT EXISTS plasma_missions (
  mission_id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS plasma_ledger_events (
  event_id TEXT PRIMARY KEY,
  mission_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  producer_type TEXT NOT NULL,
  producer_id TEXT NOT NULL,
  causation_event_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id),
  UNIQUE (mission_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_plasma_ledger_events_mission_sequence
  ON plasma_ledger_events (mission_id, sequence);
