package app

import sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"

func (tx *sqliteTx) Folder(id string) (Folder, bool) {
	row, err := tx.q.GetFolder(tx.ctx, id)
	if tx.missing(err) {
		return Folder{}, false
	}
	return sqliteFolder(row), true
}

func (tx *sqliteTx) Folders() []Folder {
	rows, err := tx.q.ListFolders(tx.ctx)
	tx.abort(err)
	folders := make([]Folder, 0, len(rows))
	for _, row := range rows {
		folders = append(folders, sqliteFolder(row))
	}
	return folders
}

func (tx *sqliteTx) PutFolder(folder Folder) {
	_, err := tx.q.UpsertFolder(tx.ctx, sqlitedb.UpsertFolderParams{
		ID: folder.ID, ParentID: sqliteNullString(folder.ParentID),
		Name: folder.Name, SystemRole: sqliteNullString(folder.SystemRole),
		SortOrder: int64(folder.SortOrder), CreatedAt: folder.CreatedAt,
		UpdatedAt: folder.UpdatedAt,
	})
	tx.abort(err)
}

func (tx *sqliteTx) DeleteFolder(id string) {
	tx.abort(tx.q.DeleteFolder(tx.ctx, id))
}
