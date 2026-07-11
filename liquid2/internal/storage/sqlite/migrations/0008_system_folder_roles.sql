ALTER TABLE folders
ADD COLUMN system_role TEXT CHECK (
  system_role IS NULL OR system_role IN ('inbox', 'trash')
);

UPDATE folders
SET system_role = 'inbox'
WHERE id = (
  SELECT id FROM folders
  WHERE parent_id IS NULL AND name = 'Inbox'
  ORDER BY CASE WHEN id = 'folder_default_inbox' THEN 0 ELSE 1 END,
    sort_order, id
  LIMIT 1
);

INSERT INTO folders (
  id, parent_id, name, sort_order, created_at, updated_at, system_role
)
SELECT
  'folder_system_trash', NULL, 'Trash', 9000,
  unixepoch() * 1000, unixepoch() * 1000, 'trash'
WHERE NOT EXISTS (
  SELECT 1 FROM folders
  WHERE parent_id IS NULL AND name = 'Trash'
);

UPDATE folders
SET system_role = 'trash'
WHERE id = (
  SELECT id FROM folders
  WHERE parent_id IS NULL AND name = 'Trash'
  ORDER BY CASE WHEN id = 'folder_system_trash' THEN 0 ELSE 1 END,
    sort_order, id
  LIMIT 1
);

CREATE UNIQUE INDEX folders_system_role_unique
  ON folders(system_role)
  WHERE system_role IS NOT NULL;
