CREATE TABLE document_versions (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  sequence INTEGER NOT NULL,
  mutation_kind TEXT NOT NULL,
  title TEXT NOT NULL,
  content_snapshot_json TEXT NOT NULL,
  metadata_snapshot_json TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  CHECK (sequence > 0),
  CHECK (mutation_kind IN ('title', 'content')),
  CHECK (length(trim(title)) > 0),
  CHECK (length(content_snapshot_json) > 0),
  CHECK (length(metadata_snapshot_json) > 0)
);

CREATE UNIQUE INDEX document_versions_document_sequence_unique
  ON document_versions(document_id, sequence);

CREATE INDEX document_versions_document_idx
  ON document_versions(document_id, created_at, id);
