-- liquid2:foreign_keys_off

CREATE TABLE folders_new (
  id TEXT PRIMARY KEY,
  parent_id TEXT REFERENCES folders_new(id) ON DELETE RESTRICT,
  name TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  system_role TEXT CHECK (
    system_role IS NULL OR system_role IN ('inbox', 'feeds', 'trash')
  ),
  CHECK (length(trim(name)) > 0)
);

INSERT INTO folders_new (
  id, parent_id, name, sort_order, created_at, updated_at, system_role
)
SELECT
  id, parent_id, name, sort_order, created_at, updated_at, system_role
FROM folders;

DROP TABLE folders;

ALTER TABLE folders_new RENAME TO folders;

CREATE UNIQUE INDEX folders_root_name_unique
  ON folders(name)
  WHERE parent_id IS NULL;

CREATE UNIQUE INDEX folders_sibling_name_unique
  ON folders(parent_id, name)
  WHERE parent_id IS NOT NULL;

CREATE UNIQUE INDEX folders_system_role_unique
  ON folders(system_role)
  WHERE system_role IS NOT NULL;

INSERT INTO folders (
  id, parent_id, name, sort_order, created_at, updated_at, system_role
)
SELECT
  'folder_system_feeds', NULL, 'Feeds', 1000,
  unixepoch() * 1000, unixepoch() * 1000, 'feeds'
WHERE NOT EXISTS (
  SELECT 1 FROM folders
  WHERE parent_id IS NULL AND name = 'Feeds'
);

UPDATE folders
SET system_role = 'feeds'
WHERE id = (
  SELECT id FROM folders
  WHERE parent_id IS NULL AND name = 'Feeds'
  ORDER BY CASE WHEN id = 'folder_system_feeds' THEN 0 ELSE 1 END,
    sort_order, id
  LIMIT 1
);
