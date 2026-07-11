package app

const (
	FolderSystemRoleInbox = "inbox"
	FolderSystemRoleFeeds = "feeds"
	FolderSystemRoleTrash = "trash"

	feedsSystemFolderID     = "folder_system_feeds"
	feedsSystemFolderName   = "Feeds"
	trashDocumentFolderID   = "folder_system_trash"
	trashDocumentFolderName = "Trash"
)

func ensureSystemFolders(tx RepositoryTx) {
	ensureDefaultDocumentFolder(tx)
	ensureFeedsFolder(tx)
	ensureTrashFolder(tx)
}

func ensureFeedsFolder(tx RepositoryTx) string {
	return ensureRootSystemFolder(tx, FolderSystemRoleFeeds, feedsSystemFolderID, feedsSystemFolderName, 1000)
}

func ensureTrashFolder(tx RepositoryTx) string {
	return ensureRootSystemFolder(tx, FolderSystemRoleTrash, trashDocumentFolderID, trashDocumentFolderName, 9000)
}

func ensureRootSystemFolder(tx RepositoryTx, role string, id string, name string, sortOrder int) string {
	if folder, ok := systemFolder(tx, role); ok {
		return folder.ID
	}
	if folder, ok := tx.Folder(id); ok {
		folder.ParentID = nil
		folder.Name = name
		folder.SystemRole = folderSystemRolePtr(role)
		tx.PutFolder(folder)
		return folder.ID
	}
	for _, folder := range tx.Folders() {
		if folder.ParentID == nil && folder.Name == name {
			folder.SystemRole = folderSystemRolePtr(role)
			tx.PutFolder(folder)
			return folder.ID
		}
	}
	now := tx.Now()
	tx.PutFolder(Folder{
		ID: id, Name: name, SystemRole: folderSystemRolePtr(role),
		SortOrder: sortOrder, CreatedAt: now, UpdatedAt: now, Children: []Folder{},
	})
	return id
}

func systemFolder(tx RepositoryReader, role string) (Folder, bool) {
	for _, folder := range tx.Folders() {
		if folder.SystemRole != nil && *folder.SystemRole == role {
			return folder, true
		}
	}
	return Folder{}, false
}

func isSystemFolder(folder Folder) bool {
	return folder.SystemRole != nil && *folder.SystemRole != ""
}

func folderHasSystemRoleAncestor(tx RepositoryReader, folderID string, role string) bool {
	for seen := map[string]struct{}{}; folderID != ""; {
		if _, ok := seen[folderID]; ok {
			return false
		}
		seen[folderID] = struct{}{}
		folder, ok := tx.Folder(folderID)
		if !ok {
			return false
		}
		if folder.SystemRole != nil && *folder.SystemRole == role {
			return true
		}
		if folder.ParentID == nil {
			return false
		}
		folderID = *folder.ParentID
	}
	return false
}

func folderHasAncestor(tx RepositoryReader, folderID string, ancestorID string) bool {
	for seen := map[string]struct{}{}; folderID != ""; {
		if folderID == ancestorID {
			return true
		}
		if _, ok := seen[folderID]; ok {
			return false
		}
		seen[folderID] = struct{}{}
		folder, ok := tx.Folder(folderID)
		if !ok || folder.ParentID == nil {
			return false
		}
		folderID = *folder.ParentID
	}
	return false
}

func folderSystemRolePtr(value string) *string {
	return &value
}
