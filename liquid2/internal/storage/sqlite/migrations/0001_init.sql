CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at INTEGER NOT NULL
);

CREATE TABLE folders (
  id TEXT PRIMARY KEY,
  parent_id TEXT REFERENCES folders(id) ON DELETE RESTRICT,
  name TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  CHECK (length(trim(name)) > 0)
);

CREATE UNIQUE INDEX folders_root_name_unique
  ON folders(name)
  WHERE parent_id IS NULL;

CREATE UNIQUE INDEX folders_sibling_name_unique
  ON folders(parent_id, name)
  WHERE parent_id IS NOT NULL;

CREATE TABLE documents (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  kind TEXT NOT NULL,
  folder_id TEXT REFERENCES folders(id) ON DELETE SET NULL,
  canonical_url TEXT,
  source_url TEXT,
  language TEXT,
  status TEXT NOT NULL,
  rating INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  read_at INTEGER,
  deleted_at INTEGER,
  CHECK (length(trim(title)) > 0),
  CHECK (kind IN ('bookmark', 'scraped_article', 'uploaded_file', 'rss_item')),
  CHECK (status IN ('unread', 'read')),
  CHECK (rating IS NULL OR (rating BETWEEN 1 AND 5))
);

CREATE INDEX documents_status_idx ON documents(status);
CREATE INDEX documents_folder_idx ON documents(folder_id);
CREATE INDEX documents_kind_idx ON documents(kind);
CREATE INDEX documents_rating_idx ON documents(rating);
CREATE INDEX documents_created_at_idx ON documents(created_at);
CREATE INDEX documents_deleted_at_idx ON documents(deleted_at);
CREATE INDEX documents_read_at_idx ON documents(read_at);

CREATE TABLE document_contents (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  format TEXT NOT NULL,
  language TEXT,
  content TEXT NOT NULL,
  source_content_id TEXT REFERENCES document_contents(id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  CHECK (role IN ('original', 'extracted', 'translation', 'summary')),
  CHECK (format IN ('html', 'markdown', 'text', 'pdf_text')),
  CHECK (length(content) > 0)
);

CREATE INDEX document_contents_document_idx ON document_contents(document_id);

CREATE TABLE document_notes (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  format TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  deleted_at INTEGER,
  CHECK (format IN ('text', 'markdown')),
  CHECK (length(trim(body)) > 0)
);

CREATE INDEX document_notes_document_idx ON document_notes(document_id, deleted_at);

CREATE TABLE tags (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  created_at INTEGER NOT NULL,
  CHECK (length(trim(name)) > 0),
  CHECK (length(trim(slug)) > 0)
);

CREATE TABLE document_tags (
  document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (document_id, tag_id)
);

CREATE TABLE blobs (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  filename TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size INTEGER NOT NULL,
  sha256 TEXT NOT NULL,
  data BLOB NOT NULL,
  created_at INTEGER NOT NULL,
  CHECK (length(trim(filename)) > 0),
  CHECK (length(trim(mime_type)) > 0),
  CHECK (size BETWEEN 0 AND 1048576),
  CHECK (size = length(data)),
  CHECK (length(data) <= 1048576),
  CHECK (length(sha256) = 64)
);

CREATE INDEX blobs_document_idx ON blobs(document_id);

CREATE TABLE feeds (
  id TEXT PRIMARY KEY,
  url TEXT NOT NULL UNIQUE,
  title TEXT,
  folder_id TEXT REFERENCES folders(id) ON DELETE SET NULL,
  enabled INTEGER NOT NULL,
  last_checked_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  CHECK (enabled IN (0, 1)),
  CHECK (length(trim(url)) > 0)
);

CREATE INDEX feeds_folder_idx ON feeds(folder_id);

CREATE TABLE feed_items (
  id TEXT PRIMARY KEY,
  feed_id TEXT NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
  document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  guid TEXT,
  url TEXT NOT NULL,
  canonical_url TEXT,
  content_hash TEXT,
  published_at INTEGER,
  created_at INTEGER NOT NULL,
  CHECK (length(trim(url)) > 0)
);

CREATE UNIQUE INDEX feed_items_guid_unique
  ON feed_items(feed_id, guid)
  WHERE guid IS NOT NULL;

CREATE UNIQUE INDEX feed_items_canonical_url_unique
  ON feed_items(feed_id, canonical_url)
  WHERE canonical_url IS NOT NULL;

CREATE UNIQUE INDEX feed_items_content_hash_unique
  ON feed_items(feed_id, content_hash)
  WHERE content_hash IS NOT NULL;

CREATE UNIQUE INDEX feed_items_url_unique ON feed_items(feed_id, url);

CREATE INDEX feed_items_feed_idx ON feed_items(feed_id);
CREATE INDEX feed_items_document_idx ON feed_items(document_id);

CREATE TABLE jobs (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  status TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  error TEXT,
  attempts INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  started_at INTEGER,
  finished_at INTEGER,
  CHECK (kind IN ('scrape_url', 'translate_document', 'poll_feed', 'extract_upload_text')),
  CHECK (status IN ('queued', 'running', 'completed', 'failed')),
  CHECK (attempts >= 0),
  CHECK (length(payload_json) > 0)
);

CREATE INDEX jobs_status_idx ON jobs(status);
CREATE INDEX jobs_kind_idx ON jobs(kind);
