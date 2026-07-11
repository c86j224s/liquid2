package app

import (
	"context"
	"log/slog"
	"sort"
	"strings"
)

// FolderInput contains folder fields accepted from API requests.
type FolderInput struct {
	// ParentID moves the folder under a parent when set.
	ParentID *string
	// Name is the folder display name.
	Name string
	// SortOrder controls sibling ordering.
	SortOrder int
}

func (s *Service) ListFolders(ctx context.Context) ([]Folder, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) ([]Folder, error) {
		ensureSystemFolders(tx)
		return folderTrees(tx.Folders()), nil
	})
}

func (s *Service) CreateFolder(ctx context.Context, input FolderInput) (Folder, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (Folder, error) {
		if err := validateFolderInput(tx, "", input); err != nil {
			return Folder{}, err
		}
		now := tx.Now()
		id := tx.NextID("folder")
		folder := Folder{
			ID: id, ParentID: cloneString(input.ParentID), Name: strings.TrimSpace(input.Name),
			SortOrder: input.SortOrder, CreatedAt: now, UpdatedAt: now, Children: []Folder{},
		}
		tx.PutFolder(folder)
		s.logger.DebugContext(ctx, "folder created", slog.String("operation", "folder_create"), slog.String("folder_id", id))
		return folderTree(newFolderTreeIndex(tx.Folders()), id), nil
	})
}

func (s *Service) UpdateFolder(ctx context.Context, id string, input FolderInput) (Folder, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (Folder, error) {
		folder, ok := tx.Folder(id)
		if !ok {
			return Folder{}, notFound("folder")
		}
		if isSystemFolder(folder) {
			return Folder{}, validation("system folder cannot be edited")
		}
		if err := validateFolderInput(tx, id, input); err != nil {
			return Folder{}, err
		}
		folder.ParentID = cloneString(input.ParentID)
		folder.Name = strings.TrimSpace(input.Name)
		folder.SortOrder = input.SortOrder
		folder.UpdatedAt = tx.Now()
		tx.PutFolder(folder)
		s.logger.DebugContext(ctx, "folder updated", slog.String("operation", "folder_update"), slog.String("folder_id", id))
		return folderTree(newFolderTreeIndex(tx.Folders()), id), nil
	})
}

func (s *Service) DeleteFolder(ctx context.Context, id string, action string) error {
	_, err := withUpdate(ctx, s, func(tx RepositoryTx) (struct{}, error) {
		folder, ok := tx.Folder(id)
		if !ok {
			return struct{}{}, notFound("folder")
		}
		if isSystemFolder(folder) {
			return struct{}{}, conflict("system folder cannot be deleted")
		}
		for _, folder := range tx.Folders() {
			if folder.ParentID != nil && *folder.ParentID == id {
				return struct{}{}, conflict("folder has child folders")
			}
		}
		now := tx.Now()
		if err := moveDeletedFolderDocuments(tx, folder, id, action, now); err != nil {
			return struct{}{}, err
		}
		for _, feed := range tx.Feeds() {
			if feed.FolderID == nil || *feed.FolderID != id {
				continue
			}
			switch action {
			case "move_to_parent":
				feed.FolderID = cloneString(folder.ParentID)
			case "move_to_uncategorized":
				feed.FolderID = nil
			default:
				return struct{}{}, conflict("folder has feeds")
			}
			feed.UpdatedAt = now
			tx.PutFeed(feed)
		}
		tx.DeleteFolder(id)
		s.logger.DebugContext(ctx, "folder deleted", slog.String("operation", "folder_delete"), slog.String("folder_id", id))
		return struct{}{}, nil
	})
	return err
}

func validateFolderInput(tx RepositoryReader, id string, input FolderInput) error {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return validation("folder name is required")
	}
	if input.ParentID != nil {
		if *input.ParentID == id {
			return validation("folder cannot be its own parent")
		}
		if _, ok := tx.Folder(*input.ParentID); !ok {
			return notFound("parent folder")
		}
		if id != "" && folderHasAncestor(tx, *input.ParentID, id) {
			return validation("folder cannot be placed under its descendant")
		}
		if parent, _ := tx.Folder(*input.ParentID); parent.SystemRole != nil && *parent.SystemRole == FolderSystemRoleTrash {
			return validation("folder cannot be placed under trash")
		}
	}
	for _, folder := range tx.Folders() {
		if folder.ID == id || folder.Name != name {
			continue
		}
		if sameOptionalString(folder.ParentID, input.ParentID) {
			return conflict("folder name must be unique among siblings")
		}
	}
	return nil
}

func folderTrees(folders []Folder) []Folder {
	index := newFolderTreeIndex(folders)
	roots := make([]Folder, 0, len(index.roots))
	for _, folder := range index.roots {
		roots = append(roots, folderTree(index, folder.ID))
	}
	sortFolders(roots)
	return roots
}

type folderTreeIndex struct {
	byID     map[string]Folder
	children map[string][]Folder
	roots    []Folder
}

func newFolderTreeIndex(folders []Folder) folderTreeIndex {
	index := folderTreeIndex{byID: map[string]Folder{}, children: map[string][]Folder{}}
	for _, folder := range folders {
		index.byID[folder.ID] = folder
		if folder.ParentID == nil {
			index.roots = append(index.roots, folder)
			continue
		}
		index.children[*folder.ParentID] = append(index.children[*folder.ParentID], folder)
	}
	return index
}

func folderTree(index folderTreeIndex, id string) Folder {
	folder := cloneFolder(index.byID[id])
	folder.Children = []Folder{}
	for _, child := range index.children[id] {
		folder.Children = append(folder.Children, folderTree(index, child.ID))
	}
	sortFolders(folder.Children)
	return folder
}

func sortFolders(folders []Folder) {
	sort.Slice(folders, func(i int, j int) bool {
		if folders[i].SortOrder != folders[j].SortOrder {
			return folders[i].SortOrder < folders[j].SortOrder
		}
		if folders[i].Name != folders[j].Name {
			return folders[i].Name < folders[j].Name
		}
		return folders[i].ID < folders[j].ID
	})
}

func sameOptionalString(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
