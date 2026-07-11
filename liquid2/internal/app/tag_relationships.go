package app

import "strings"

func normalizeTagIDs(tx RepositoryReader, tagIDs []string) ([]string, error) {
	ids := make([]string, 0, len(tagIDs))
	seen := map[string]struct{}{}
	for _, id := range tagIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := tx.Tag(id); !ok {
			return nil, notFound("tag")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func removedDocumentTagIDs(before []string, after []string) []string {
	kept := map[string]struct{}{}
	for _, id := range after {
		kept[id] = struct{}{}
	}
	removed := []string{}
	seen := map[string]struct{}{}
	for _, id := range before {
		if _, ok := kept[id]; ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		removed = append(removed, id)
	}
	return removed
}

func deleteUnusedTags(tx RepositoryTx, tagIDs []string) int {
	deleted := 0
	for _, id := range tagIDs {
		if _, ok := tx.Tag(id); !ok {
			continue
		}
		if tx.TagHasDocuments(id) {
			continue
		}
		tx.DeleteTag(id)
		deleted++
	}
	return deleted
}

func hasValueInSlice(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func removeStringValue(values []string, value string) []string {
	filtered := values[:0]
	for _, candidate := range values {
		if candidate != value {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}
