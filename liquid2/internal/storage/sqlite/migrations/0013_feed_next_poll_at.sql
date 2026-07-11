ALTER TABLE app_settings ADD COLUMN feed_next_poll_at INTEGER;

UPDATE app_settings
SET feed_poll_interval_seconds = 7200
WHERE feed_poll_interval_seconds = 1800
  AND updated_at = 0;
