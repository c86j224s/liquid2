package app

import (
	"sort"
	"strings"
)

const (
	DocumentSortRelevance   = "relevance"
	DocumentSortRecent      = "recent"
	DocumentSortCreatedDesc = "created_desc"
	DocumentSortRatingDesc  = "rating_desc"

	maxDocumentQueryLength = 256
)

type documentListEntry struct {
	record documentRecord
	score  int
}

func normalizeDocumentFilters(filters DocumentFilters) (DocumentFilters, error) {
	filters.Query = strings.TrimSpace(filters.Query)
	filters.Sort = strings.TrimSpace(filters.Sort)
	if len([]rune(filters.Query)) > maxDocumentQueryLength {
		return DocumentFilters{}, validation("query must be at most 256 characters")
	}
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 50
	}
	if filters.Sort == "" {
		if filters.Query == "" {
			filters.Sort = DocumentSortRecent
		} else {
			filters.Sort = DocumentSortRelevance
		}
	}
	switch filters.Sort {
	case DocumentSortRelevance:
		if filters.Query == "" {
			return DocumentFilters{}, validation("relevance sort requires query")
		}
	case DocumentSortRecent, DocumentSortCreatedDesc, DocumentSortRatingDesc:
	default:
		return DocumentFilters{}, validation("sort is invalid")
	}
	return filters, nil
}

func listDocumentIDsFromRecords(
	tx RepositoryReader,
	records []documentRecord,
	filters DocumentFilters,
) ([]string, *string, error) {
	filters, err := normalizeDocumentFilters(filters)
	if err != nil {
		return nil, nil, err
	}
	folderIDs := documentFolderFilterIDs(tx, filters)
	tokens := queryTokens(filters.Query)
	entries := make([]documentListEntry, 0, len(records))
	for _, record := range records {
		if !matchesDocumentFiltersWithFolders(tx, record, filters, folderIDs) {
			continue
		}
		score, ok := documentSearchScore(record, tokens)
		if !ok {
			continue
		}
		entries = append(entries, documentListEntry{record: record, score: score})
	}
	sortDocumentEntries(entries, filters.Sort)
	start, err := documentCursorStart(entries, filters.Cursor)
	if err != nil {
		return nil, nil, err
	}
	end := start + filters.Limit
	if end > len(entries) {
		end = len(entries)
	}
	ids := make([]string, 0, end-start)
	for _, entry := range entries[start:end] {
		ids = append(ids, entry.record.meta.ID)
	}
	var nextCursor *string
	if end < len(entries) {
		nextCursor = encodeDocumentCursor(entries[end-1].record.meta.ID)
	}
	return ids, nextCursor, nil
}

func documentFolderFilterIDs(tx RepositoryReader, filters DocumentFilters) map[string]struct{} {
	if filters.FolderID == "" {
		return nil
	}
	ids := map[string]struct{}{filters.FolderID: {}}
	if !filters.IncludeFolderDescendants {
		return ids
	}
	changed := true
	for changed {
		changed = false
		for _, folder := range tx.Folders() {
			if folder.ParentID == nil {
				continue
			}
			if _, ok := ids[*folder.ParentID]; !ok {
				continue
			}
			if _, ok := ids[folder.ID]; !ok {
				ids[folder.ID] = struct{}{}
				changed = true
			}
		}
	}
	return ids
}

func queryTokens(query string) []string {
	fields := strings.Fields(strings.ToLower(query))
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func documentSearchScore(record documentRecord, tokens []string) (int, bool) {
	if len(tokens) == 0 {
		return 0, true
	}
	title := strings.ToLower(record.meta.Title)
	body := strings.ToLower(documentSearchBody(record))
	score := 0
	for _, token := range tokens {
		tokenScore := 0
		if strings.Contains(title, token) {
			tokenScore += 100
		}
		if strings.Contains(body, token) {
			tokenScore += 10
		}
		if tokenScore == 0 {
			return 0, false
		}
		score += tokenScore
	}
	return score, true
}

func documentSearchBody(record documentRecord) string {
	var builder strings.Builder
	for _, content := range record.contents {
		builder.WriteString(content.Content)
		builder.WriteByte('\n')
	}
	return builder.String()
}

func sortDocumentEntries(entries []documentListEntry, sortValue string) {
	sort.SliceStable(entries, func(i int, j int) bool {
		left := entries[i]
		right := entries[j]
		switch sortValue {
		case DocumentSortRelevance:
			if left.score != right.score {
				return left.score > right.score
			}
			return newerDocument(left.record, right.record)
		case DocumentSortCreatedDesc:
			if left.record.meta.CreatedAt != right.record.meta.CreatedAt {
				return left.record.meta.CreatedAt > right.record.meta.CreatedAt
			}
		case DocumentSortRatingDesc:
			if documentRating(left.record) != documentRating(right.record) {
				return documentRating(left.record) > documentRating(right.record)
			}
			return newerDocument(left.record, right.record)
		default:
			return newerDocument(left.record, right.record)
		}
		return left.record.meta.ID > right.record.meta.ID
	})
}

func newerDocument(left documentRecord, right documentRecord) bool {
	if left.meta.UpdatedAt != right.meta.UpdatedAt {
		return left.meta.UpdatedAt > right.meta.UpdatedAt
	}
	if left.meta.CreatedAt != right.meta.CreatedAt {
		return left.meta.CreatedAt > right.meta.CreatedAt
	}
	return left.meta.ID > right.meta.ID
}

func documentRating(record documentRecord) int {
	if record.meta.Rating == nil {
		return 0
	}
	return *record.meta.Rating
}
