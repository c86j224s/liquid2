package app

import "strings"

const (
	defaultDocumentFolderID   = "folder_default_inbox"
	defaultDocumentFolderName = "Inbox"
)

func normalizeDocumentFolderID(tx RepositoryTx, folderID string) (*string, error) {
	return normalizeDocumentFolderIDFor(tx, folderID, false)
}

func normalizeFeedDocumentFolderID(tx RepositoryTx, folderID string) (*string, error) {
	return normalizeDocumentFolderIDFor(tx, folderID, true)
}

func normalizeDocumentFolderIDFor(tx RepositoryTx, folderID string, allowFeeds bool) (*string, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		id := ensureDefaultDocumentFolder(tx)
		return &id, nil
	}
	if _, ok := tx.Folder(folderID); !ok {
		return nil, notFound("folder")
	}
	if folderHasSystemRoleAncestor(tx, folderID, FolderSystemRoleTrash) {
		return nil, validation("folder cannot be trash")
	}
	if !allowFeeds && folderHasSystemRoleAncestor(tx, folderID, FolderSystemRoleFeeds) {
		return nil, validation("folder cannot be feeds")
	}
	return &folderID, nil
}

func ensureDefaultDocumentFolder(tx RepositoryTx) string {
	if folder, ok := systemFolder(tx, FolderSystemRoleInbox); ok {
		return folder.ID
	}
	if folder, ok := tx.Folder(defaultDocumentFolderID); ok {
		folder.SystemRole = folderSystemRolePtr(FolderSystemRoleInbox)
		tx.PutFolder(folder)
		return folder.ID
	}
	for _, folder := range tx.Folders() {
		if folder.ParentID == nil && folder.Name == defaultDocumentFolderName {
			folder.SystemRole = folderSystemRolePtr(FolderSystemRoleInbox)
			tx.PutFolder(folder)
			return folder.ID
		}
	}
	now := tx.Now()
	tx.PutFolder(Folder{
		ID: defaultDocumentFolderID, Name: defaultDocumentFolderName,
		SystemRole: folderSystemRolePtr(FolderSystemRoleInbox),
		CreatedAt:  now, UpdatedAt: now, Children: []Folder{},
	})
	return defaultDocumentFolderID
}

func defaultDocumentFolderForDelete(tx RepositoryTx, deletedFolderID string) (*string, error) {
	if folder, ok := tx.Folder(defaultDocumentFolderID); ok && folder.ID != deletedFolderID {
		id := folder.ID
		return &id, nil
	}
	for _, folder := range tx.Folders() {
		if folder.ID == deletedFolderID {
			continue
		}
		if folder.ParentID == nil && folder.Name == defaultDocumentFolderName {
			id := folder.ID
			return &id, nil
		}
	}
	if deletedFolderID == defaultDocumentFolderID {
		return nil, conflict("default folder has documents")
	}
	id := ensureDefaultDocumentFolder(tx)
	if id == deletedFolderID {
		return nil, conflict("default folder has documents")
	}
	return &id, nil
}
