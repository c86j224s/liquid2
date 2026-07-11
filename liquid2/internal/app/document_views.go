package app

import "sort"

func documentDetail(tx RepositoryReader, id string) DocumentDetail {
	record, ok := tx.Document(id)
	if !ok {
		panic("documentDetail: document not found: " + id)
	}
	return DocumentDetail{
		Document:   documentMetadata(tx, record),
		FolderPath: documentFolderPath(tx, record.meta.FolderID),
		Contents:   cloneDocumentContents(record.contents),
		Tags:       documentTags(tx, record),
		Blobs:      append([]BlobMetadata(nil), record.blobs...),
	}
}

func documentSummary(tx RepositoryReader, record documentRecord) DocumentSummary {
	meta := documentMetadata(tx, record)
	tags := documentTags(tx, record)
	slugs := make([]string, 0, len(tags))
	for _, tag := range tags {
		slugs = append(slugs, tag.Slug)
	}
	return DocumentSummary{
		ID: meta.ID, Title: meta.Title, Kind: meta.Kind, FolderID: cloneString(meta.FolderID),
		FolderPath:   documentFolderPath(tx, meta.FolderID),
		CanonicalURL: cloneString(meta.CanonicalURL), SourceURL: cloneString(meta.SourceURL),
		Language: cloneString(meta.Language), Status: meta.Status, Rating: cloneInt(meta.Rating),
		CreatedAt: meta.CreatedAt, UpdatedAt: meta.UpdatedAt, PublishedAt: cloneInt64(meta.PublishedAt),
		ReadAt:    cloneInt64(meta.ReadAt),
		DeletedAt: cloneInt64(meta.DeletedAt), Tags: slugs,
	}
}

func documentMetadata(tx RepositoryReader, record documentRecord) DocumentMetadata {
	meta := cloneDocumentMetadata(record.meta)
	if item, ok := tx.FeedItemByDocumentID(meta.ID); ok {
		meta.PublishedAt = cloneInt64(item.PublishedAt)
	}
	return meta
}

func documentTags(tx RepositoryReader, record documentRecord) []Tag {
	tags := make([]Tag, 0, len(record.tagIDs))
	for _, id := range record.tagIDs {
		if tag, ok := tx.Tag(id); ok {
			tags = append(tags, tag)
		}
	}
	sort.Slice(tags, func(i int, j int) bool {
		if tags[i].Slug != tags[j].Slug {
			return tags[i].Slug < tags[j].Slug
		}
		return tags[i].ID < tags[j].ID
	})
	return tags
}

func matchesDocumentFilters(tx RepositoryReader, record documentRecord, filters DocumentFilters) bool {
	return matchesDocumentFiltersWithFolders(tx, record, filters, documentFolderFilterIDs(tx, filters))
}

func matchesDocumentFiltersWithFolders(
	tx RepositoryReader,
	record documentRecord,
	filters DocumentFilters,
	folderIDs map[string]struct{},
) bool {
	meta := record.meta
	if !filters.IncludeDeleted && meta.DeletedAt != nil {
		return false
	}
	if !filters.IncludeTrash && filters.FolderID == "" && isTrashDocument(tx, meta) {
		return false
	}
	if filters.Status != "" && meta.Status != filters.Status {
		return false
	}
	if len(folderIDs) > 0 && (meta.FolderID == nil || !hasStringKey(folderIDs, *meta.FolderID)) {
		return false
	}
	if filters.Kind != "" && meta.Kind != filters.Kind {
		return false
	}
	if filters.RatingMin > 0 && (meta.Rating == nil || *meta.Rating < filters.RatingMin) {
		return false
	}
	if filters.Tag != "" && !hasTagSlug(tx, record.tagIDs, filters.Tag) {
		return false
	}
	return true
}

func isTrashDocument(tx RepositoryReader, meta DocumentMetadata) bool {
	if meta.FolderID == nil {
		return false
	}
	return folderHasSystemRoleAncestor(tx, *meta.FolderID, FolderSystemRoleTrash)
}

func hasStringKey(values map[string]struct{}, value string) bool {
	_, ok := values[value]
	return ok
}

func hasTagSlug(tx RepositoryReader, tagIDs []string, slug string) bool {
	for _, id := range tagIDs {
		if tag, ok := tx.Tag(id); ok && tag.Slug == slug {
			return true
		}
	}
	return false
}

func cloneDocumentRecord(record documentRecord) documentRecord {
	blobData := record.blobData
	record.meta = cloneDocumentMetadata(record.meta)
	record.contents = cloneDocumentContents(record.contents)
	record.blobs = append([]BlobMetadata(nil), record.blobs...)
	record.tagIDs = append([]string(nil), record.tagIDs...)
	record.blobData = map[string][]byte{}
	for id, data := range blobData {
		record.blobData[id] = append([]byte(nil), data...)
	}
	return record
}

func cloneDocumentContents(contents []DocumentContent) []DocumentContent {
	if contents == nil {
		return nil
	}
	cloned := make([]DocumentContent, len(contents))
	for i, content := range contents {
		cloned[i] = cloneDocumentContent(content)
	}
	return cloned
}

func cloneDocumentContent(content DocumentContent) DocumentContent {
	content.Language = cloneString(content.Language)
	content.SourceContentID = cloneString(content.SourceContentID)
	return content
}

func cloneFolder(folder Folder) Folder {
	folder.ParentID = cloneString(folder.ParentID)
	folder.SystemRole = cloneString(folder.SystemRole)
	folder.Children = append([]Folder(nil), folder.Children...)
	return folder
}

func cloneDocumentMetadata(meta DocumentMetadata) DocumentMetadata {
	meta.FolderID = cloneString(meta.FolderID)
	meta.CanonicalURL = cloneString(meta.CanonicalURL)
	meta.SourceURL = cloneString(meta.SourceURL)
	meta.Language = cloneString(meta.Language)
	meta.Rating = cloneInt(meta.Rating)
	meta.PublishedAt = cloneInt64(meta.PublishedAt)
	meta.ReadAt = cloneInt64(meta.ReadAt)
	meta.DeletedAt = cloneInt64(meta.DeletedAt)
	return meta
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
