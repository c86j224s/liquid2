INSERT INTO folders (
  id, parent_id, name, sort_order, created_at, updated_at
)
SELECT
  'folder_default_inbox', NULL, 'Inbox', 0,
  unixepoch() * 1000, unixepoch() * 1000
WHERE NOT EXISTS (
  SELECT 1 FROM folders
  WHERE parent_id IS NULL AND name = 'Inbox'
)
AND NOT EXISTS (
  SELECT 1 FROM folders WHERE id = 'folder_default_inbox'
);

UPDATE documents
SET folder_id = COALESCE(
  (SELECT id FROM folders WHERE id = 'folder_default_inbox'),
  (
    SELECT id FROM folders
    WHERE parent_id IS NULL AND name = 'Inbox'
    ORDER BY sort_order, name, id
    LIMIT 1
  )
)
WHERE folder_id IS NULL;

CREATE TRIGGER documents_folder_required_insert
BEFORE INSERT ON documents
WHEN NEW.folder_id IS NULL OR length(trim(NEW.folder_id)) = 0
BEGIN
  SELECT RAISE(ABORT, 'document folder is required');
END;

CREATE TRIGGER documents_folder_required_update
BEFORE UPDATE OF folder_id ON documents
WHEN NEW.folder_id IS NULL OR length(trim(NEW.folder_id)) = 0
BEGIN
  SELECT RAISE(ABORT, 'document folder is required');
END;
