CREATE TABLE app_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  feed_scheduler_enabled INTEGER NOT NULL DEFAULT 0,
  feed_poll_interval_seconds INTEGER NOT NULL DEFAULT 7200,
  updated_at INTEGER NOT NULL DEFAULT 0,
  CHECK (feed_scheduler_enabled IN (0, 1)),
  CHECK (feed_poll_interval_seconds BETWEEN 60 AND 86400)
);

INSERT INTO app_settings (
  id, feed_scheduler_enabled, feed_poll_interval_seconds, updated_at
) VALUES (1, 0, 7200, 0);
