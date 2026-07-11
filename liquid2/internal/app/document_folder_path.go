package app

func documentFolderPath(tx RepositoryReader, folderID *string) []FolderBreadcrumb {
	if folderID == nil {
		return []FolderBreadcrumb{}
	}
	var folders []Folder
	seen := map[string]struct{}{}
	for id := *folderID; id != ""; {
		if _, ok := seen[id]; ok {
			break
		}
		seen[id] = struct{}{}
		folder, ok := tx.Folder(id)
		if !ok {
			break
		}
		folders = append(folders, folder)
		if folder.ParentID == nil {
			break
		}
		id = *folder.ParentID
	}
	path := make([]FolderBreadcrumb, 0, len(folders))
	for index := len(folders) - 1; index >= 0; index-- {
		path = append(path, FolderBreadcrumb{
			ID: folders[index].ID, Name: folders[index].Name,
		})
	}
	return path
}
