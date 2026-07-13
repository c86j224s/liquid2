CREATE INDEX IF NOT EXISTS idx_plasma_ledger_events_activity_list
  ON plasma_ledger_events (mission_id, event_type, sequence);
