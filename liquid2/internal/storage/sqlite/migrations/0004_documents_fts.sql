CREATE VIRTUAL TABLE documents_fts USING fts5(
  document_id UNINDEXED,
  title,
  body,
  tokenize = 'unicode61 remove_diacritics 2'
);

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
