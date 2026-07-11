-- liquid2:foreign_keys_off

DROP TRIGGER IF EXISTS documents_fts_insert;
DROP TRIGGER IF EXISTS documents_fts_update;
DROP TRIGGER IF EXISTS documents_fts_delete;
DROP TRIGGER IF EXISTS document_contents_fts_insert;
DROP TRIGGER IF EXISTS document_contents_fts_update;
DROP TRIGGER IF EXISTS document_contents_fts_delete;
DROP TRIGGER IF EXISTS documents_folder_required_insert;
DROP TRIGGER IF EXISTS documents_folder_required_update;

CREATE TABLE documents_new (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  kind TEXT NOT NULL,
  folder_id TEXT REFERENCES folders(id) ON DELETE RESTRICT,
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

INSERT INTO documents_new (
  id, title, kind, folder_id, canonical_url, source_url, language, status,
  rating, created_at, updated_at, read_at, deleted_at
)
SELECT
  id, title, kind, folder_id, canonical_url, source_url, language, status,
  rating, created_at, updated_at, read_at, deleted_at
FROM documents;

DROP TABLE documents;

ALTER TABLE documents_new RENAME TO documents;

CREATE INDEX documents_status_idx ON documents(status);
CREATE INDEX documents_folder_idx ON documents(folder_id);
CREATE INDEX documents_kind_idx ON documents(kind);
CREATE INDEX documents_rating_idx ON documents(rating);
CREATE INDEX documents_created_at_idx ON documents(created_at);
CREATE INDEX documents_deleted_at_idx ON documents(deleted_at);
CREATE INDEX documents_read_at_idx ON documents(read_at);

CREATE INDEX documents_recent_order_idx
  ON documents(updated_at DESC, created_at DESC, id DESC);

CREATE INDEX documents_created_order_idx
  ON documents(created_at DESC, id DESC);

CREATE INDEX documents_rating_order_idx
  ON documents(COALESCE(rating, 0) DESC, updated_at DESC, created_at DESC, id DESC);

CREATE TRIGGER documents_folder_required_insert
BEFORE INSERT ON documents
WHEN NEW.folder_id IS NULL OR length(trim(NEW.folder_id)) = 0
BEGIN
  SELECT RAISE(ABORT, 'document folder_id is required');
END;

CREATE TRIGGER documents_folder_required_update
BEFORE UPDATE OF folder_id ON documents
WHEN NEW.folder_id IS NULL OR length(trim(NEW.folder_id)) = 0
BEGIN
  SELECT RAISE(ABORT, 'document folder_id is required');
END;

CREATE TRIGGER documents_fts_insert
AFTER INSERT ON documents
BEGIN
  INSERT INTO documents_fts(document_id, title, body)
  VALUES (
    new.id,
    new.title,
    COALESCE((
      SELECT group_concat(content, char(10))
      FROM document_contents
      WHERE document_id = new.id
    ), '')
  );
END;

CREATE TRIGGER documents_fts_update
AFTER UPDATE OF title ON documents
BEGIN
  DELETE FROM documents_fts WHERE document_id = old.id;
  INSERT INTO documents_fts(document_id, title, body)
  VALUES (
    new.id,
    new.title,
    COALESCE((
      SELECT group_concat(content, char(10))
      FROM document_contents
      WHERE document_id = new.id
    ), '')
  );
END;

CREATE TRIGGER documents_fts_delete
AFTER DELETE ON documents
BEGIN
  DELETE FROM documents_fts WHERE document_id = old.id;
END;

CREATE TRIGGER document_contents_fts_insert
AFTER INSERT ON document_contents
BEGIN
  DELETE FROM documents_fts WHERE document_id = new.document_id;
  INSERT INTO documents_fts(document_id, title, body)
  SELECT
    documents.id,
    documents.title,
    COALESCE((
      SELECT group_concat(content, char(10))
      FROM document_contents
      WHERE document_id = documents.id
    ), '')
  FROM documents
  WHERE documents.id = new.document_id;
END;

CREATE TRIGGER document_contents_fts_update
AFTER UPDATE ON document_contents
BEGIN
  DELETE FROM documents_fts WHERE document_id = old.document_id;
  DELETE FROM documents_fts WHERE document_id = new.document_id;
  INSERT INTO documents_fts(document_id, title, body)
  SELECT
    documents.id,
    documents.title,
    COALESCE((
      SELECT group_concat(content, char(10))
      FROM document_contents
      WHERE document_id = documents.id
    ), '')
  FROM documents
  WHERE documents.id IN (old.document_id, new.document_id);
END;

CREATE TRIGGER document_contents_fts_delete
AFTER DELETE ON document_contents
BEGIN
  DELETE FROM documents_fts WHERE document_id = old.document_id;
  INSERT INTO documents_fts(document_id, title, body)
  SELECT
    documents.id,
    documents.title,
    COALESCE((
      SELECT group_concat(content, char(10))
      FROM document_contents
      WHERE document_id = documents.id
    ), '')
  FROM documents
  WHERE documents.id = old.document_id;
END;

DELETE FROM documents_fts;

INSERT INTO documents_fts(document_id, title, body)
SELECT
  documents.id,
  documents.title,
  COALESCE((
    SELECT group_concat(content, char(10))
    FROM document_contents
    WHERE document_id = documents.id
  ), '')
FROM documents;
