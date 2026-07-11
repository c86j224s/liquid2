package app

func moveDeletedFolderDocuments(
	tx RepositoryTx,
	folder Folder,
	folderID string,
	action string,
	now int64,
) error {
	for _, doc := range tx.Documents() {
		if doc.meta.FolderID == nil || *doc.meta.FolderID != folderID {
			continue
		}
		target, err := deletedFolderDocumentTarget(tx, folder, folderID, action)
		if err != nil {
			return err
		}
		doc.meta.FolderID = target
		doc.meta.UpdatedAt = now
		tx.PutDocument(doc)
	}
	return nil
}

func deletedFolderDocumentTarget(
	tx RepositoryTx,
	folder Folder,
	folderID string,
	action string,
) (*string, error) {
	switch action {
	case "move_to_parent":
		if folder.ParentID != nil {
			return cloneString(folder.ParentID), nil
		}
		return defaultDocumentFolderForDelete(tx, folderID)
	case "move_to_uncategorized":
		return defaultDocumentFolderForDelete(tx, folderID)
	default:
		return nil, conflict("folder has documents")
	}
}
