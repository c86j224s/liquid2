CREATE TABLE IF NOT EXISTS plasma_evidence_records (
  evidence_id TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  object_kind TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  state TEXT NOT NULL,
  summary TEXT NOT NULL,
  evidence_type TEXT NOT NULL,
  snapshot_refs_json TEXT NOT NULL DEFAULT '[]',
  confidence_json TEXT NOT NULL DEFAULT '{}',
  producer_type TEXT NOT NULL,
  producer_id TEXT NOT NULL,
  created_event_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_evidence_records_mission_state
  ON plasma_evidence_records (mission_id, state);

CREATE TABLE IF NOT EXISTS plasma_claim_records (
  claim_id TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  object_kind TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  state TEXT NOT NULL,
  text TEXT NOT NULL,
  claim_type TEXT NOT NULL,
  supporting_evidence_ids_json TEXT NOT NULL DEFAULT '[]',
  opposing_evidence_ids_json TEXT NOT NULL DEFAULT '[]',
  depends_on_question_ids_json TEXT NOT NULL DEFAULT '[]',
  user_assertion_event_id TEXT NOT NULL DEFAULT '',
  confidence_json TEXT NOT NULL DEFAULT '{}',
  approval_json TEXT NOT NULL DEFAULT '{}',
  created_event_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_claim_records_mission_state
  ON plasma_claim_records (mission_id, state);

CREATE TABLE IF NOT EXISTS plasma_question_records (
  question_id TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  object_kind TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  state TEXT NOT NULL,
  text TEXT NOT NULL,
  priority TEXT NOT NULL,
  blocking INTEGER NOT NULL DEFAULT 0,
  related_evidence_ids_json TEXT NOT NULL DEFAULT '[]',
  related_claim_ids_json TEXT NOT NULL DEFAULT '[]',
  resolution TEXT NOT NULL DEFAULT '',
  created_event_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_question_records_mission_state
  ON plasma_question_records (mission_id, state);

CREATE TABLE IF NOT EXISTS plasma_option_records (
  option_id TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  object_kind TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  state TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  pros_json TEXT NOT NULL DEFAULT '[]',
  cons_json TEXT NOT NULL DEFAULT '[]',
  supporting_claim_ids_json TEXT NOT NULL DEFAULT '[]',
  risk_level TEXT NOT NULL,
  created_event_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_option_records_mission_state
  ON plasma_option_records (mission_id, state);

CREATE TABLE IF NOT EXISTS plasma_proposal_bundles (
  proposal_id TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  object_kind TEXT NOT NULL,
  mission_id TEXT NOT NULL,
  state TEXT NOT NULL,
  title TEXT NOT NULL,
  object_refs_json TEXT NOT NULL DEFAULT '[]',
  requested_decision TEXT NOT NULL,
  created_event_id TEXT NOT NULL,
  decision_event_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  decided_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  FOREIGN KEY (mission_id) REFERENCES plasma_missions (mission_id)
);

CREATE INDEX IF NOT EXISTS idx_plasma_proposal_bundles_mission_state
  ON plasma_proposal_bundles (mission_id, state);
