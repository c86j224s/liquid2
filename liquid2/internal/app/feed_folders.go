package app

import (
	"net/url"
	"path"
	"strconv"
	"strings"
)

func createFeedDocumentFolder(tx RepositoryTx, title *string, feedURL string) string {
	parentID := ensureFeedsFolder(tx)
	name := uniqueFeedFolderName(tx, parentID, feedFolderBaseName(title, feedURL))
	now := tx.Now()
	id := tx.NextID("folder")
	tx.PutFolder(Folder{
		ID: id, ParentID: &parentID, Name: name,
		SortOrder: nextChildFolderSortOrder(tx, parentID),
		CreatedAt: now, UpdatedAt: now, Children: []Folder{},
	})
	return id
}

func feedFolderBaseName(title *string, feedURL string) string {
	if title != nil {
		if value := strings.TrimSpace(*title); value != "" {
			return value
		}
	}
	parsed, err := url.Parse(feedURL)
	if err != nil || parsed.Hostname() == "" {
		return "Feed"
	}
	host := strings.ToLower(parsed.Hostname())
	last := strings.TrimSpace(path.Base(parsed.Path))
	last = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(last, ".xml"), ".rss"), ".atom")
	last = strings.ReplaceAll(strings.ReplaceAll(last, "-", " "), "_", " ")
	if last == "" || last == "." || last == "/" {
		return host
	}
	return strings.TrimSpace(host + " " + last)
}

func uniqueFeedFolderName(tx RepositoryReader, parentID string, base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "Feed"
	}
	used := map[string]struct{}{}
	for _, folder := range tx.Folders() {
		if folder.ParentID != nil && *folder.ParentID == parentID {
			used[folder.Name] = struct{}{}
		}
	}
	if _, ok := used[base]; !ok {
		return base
	}
	for suffix := 2; ; suffix++ {
		candidate := base + " (" + strconv.Itoa(suffix) + ")"
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
}

func nextChildFolderSortOrder(tx RepositoryReader, parentID string) int {
	next := 10
	for _, folder := range tx.Folders() {
		if folder.ParentID != nil && *folder.ParentID == parentID && folder.SortOrder >= next {
			next = folder.SortOrder + 10
		}
	}
	return next
}
