DROP INDEX IF EXISTS idx_plasma_raw_artifacts_mission_sha;

CREATE INDEX IF NOT EXISTS idx_plasma_raw_artifacts_mission_sha
  ON plasma_raw_artifacts (mission_id, sha256);
