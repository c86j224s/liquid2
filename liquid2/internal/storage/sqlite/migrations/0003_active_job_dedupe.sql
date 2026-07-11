UPDATE jobs
SET status = 'failed',
    error = 'deduplicated active job during migration',
    updated_at = unixepoch() * 1000,
    finished_at = unixepoch() * 1000
WHERE id IN (
  SELECT id FROM (
    SELECT id,
           row_number() OVER (
             PARTITION BY kind, payload_json
             ORDER BY created_at ASC, id ASC
           ) AS duplicate_rank
    FROM jobs
    WHERE status IN ('queued', 'running')
  )
  WHERE duplicate_rank > 1
);

CREATE UNIQUE INDEX jobs_active_payload_unique
  ON jobs(kind, payload_json)
  WHERE status IN ('queued', 'running');
