package app

import "sort"

type orphanFeedDocumentIndex struct {
	records           []documentRecord
	linkedDocumentIDs map[string]struct{}
}

func newOrphanFeedDocumentIndex(tx RepositoryReader) *orphanFeedDocumentIndex {
	records := tx.Documents()
	sort.Slice(records, func(i int, j int) bool {
		left := records[i].meta
		right := records[j].meta
		if left.CreatedAt != right.CreatedAt {
			return left.CreatedAt < right.CreatedAt
		}
		return left.ID < right.ID
	})
	return &orphanFeedDocumentIndex{
		records:           records,
		linkedDocumentIDs: linkedFeedDocumentIDs(tx),
	}
}

func (index *orphanFeedDocumentIndex) adopt(tx RepositoryTx, feed Feed, item normalizedFeedItem) (string, bool) {
	for i, record := range index.records {
		if !record.matchesOrphanFeedDocument(item) {
			continue
		}
		if _, ok := index.linkedDocumentIDs[record.meta.ID]; ok {
			continue
		}
		index.linkedDocumentIDs[record.meta.ID] = struct{}{}
		if shouldMoveAdoptedFeedDocument(tx, record, feed) {
			record.meta.FolderID = cloneString(feed.FolderID)
			record.meta.UpdatedAt = tx.Now()
			tx.PutDocument(record)
			index.records[i] = record
		}
		return record.meta.ID, true
	}
	return "", false
}

func (record documentRecord) matchesOrphanFeedDocument(item normalizedFeedItem) bool {
	if record.meta.Kind != DocumentKindRSSItem || record.meta.DeletedAt != nil {
		return false
	}
	return feedDocumentURLMatches(record.meta.CanonicalURL, item) ||
		feedDocumentURLMatches(record.meta.SourceURL, item)
}

func feedDocumentURLMatches(value *string, item normalizedFeedItem) bool {
	if value == nil || *value == "" {
		return false
	}
	if item.canonicalURL != nil && *value == *item.canonicalURL {
		return true
	}
	if item.sourceURL != nil && *value == *item.sourceURL {
		return true
	}
	return *value == item.url
}

func shouldMoveAdoptedFeedDocument(tx RepositoryReader, record documentRecord, feed Feed) bool {
	if feed.FolderID == nil {
		return false
	}
	if record.meta.FolderID == nil {
		return true
	}
	if *record.meta.FolderID == *feed.FolderID {
		return false
	}
	return !folderHasSystemRoleAncestor(tx, *record.meta.FolderID, FolderSystemRoleTrash)
}

func linkedFeedDocumentIDs(tx RepositoryReader) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, feed := range tx.Feeds() {
		for _, item := range tx.FeedItems(feed.ID) {
			ids[item.DocumentID] = struct{}{}
		}
	}
	return ids
}
