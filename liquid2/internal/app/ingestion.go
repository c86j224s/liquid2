package app

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
)

func (s *Service) CreateBookmarkDocument(ctx context.Context, input BookmarkDocumentInput) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		spec, err := ingestSpec(tx, DocumentKindBookmark, input.URL, input.SourceURL, input.Title, input.FolderID, input.TagIDs)
		if err != nil {
			return DocumentDetail{}, err
		}
		id := createIngestedDocument(tx, spec)
		s.logger.DebugContext(ctx, "bookmark document created",
			slog.String("operation", "document_bookmark"),
			slog.String("document_id", id),
		)
		return documentDetail(tx, id), nil
	})
}

func (s *Service) CreateScrapedDocument(ctx context.Context, input ScrapedDocumentInput) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		spec, err := ingestSpec(tx, DocumentKindScrapedArticle, input.URL, input.SourceURL, input.Title, input.FolderID, input.TagIDs)
		if err != nil {
			return DocumentDetail{}, err
		}
		content := strings.TrimSpace(input.Content)
		if content == "" {
			return DocumentDetail{}, validation("content is required")
		}
		format := strings.TrimSpace(input.Format)
		if format == "" {
			format = ContentFormatText
		}
		spec.contents = []DocumentContent{contentRecord(tx, content, format)}
		id := createIngestedDocument(tx, spec)
		s.logger.DebugContext(ctx, "scraped document created",
			slog.String("operation", "document_scrape"),
			slog.String("document_id", id),
		)
		return documentDetail(tx, id), nil
	})
}

func (s *Service) CreateUploadedDocument(ctx context.Context, input UploadedDocumentInput) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		title := input.Title
		if strings.TrimSpace(title) == "" {
			title = input.Filename
		}
		spec, err := ingestSpec(tx, DocumentKindUploadedFile, "", "", title, input.FolderID, input.TagIDs)
		if err != nil {
			return DocumentDetail{}, err
		}
		filename := strings.TrimSpace(filepath.Base(input.Filename))
		mimeType := strings.TrimSpace(input.MimeType)
		if filename == "" || filename == "." {
			return DocumentDetail{}, validation("filename is required")
		}
		if mimeType == "" {
			return DocumentDetail{}, validation("mime type is required")
		}
		if len(input.Data) == 0 {
			return DocumentDetail{}, validation("file data is required")
		}
		blobID := tx.NextID("blob")
		spec.blobs = []BlobMetadata{{
			ID: blobID, Filename: filename, MimeType: mimeType,
			Size: int64(len(input.Data)), CreatedAt: tx.Now(),
		}}
		spec.blobData = map[string][]byte{blobID: append([]byte(nil), input.Data...)}
		if content := strings.TrimSpace(input.Content); content != "" {
			format := strings.TrimSpace(input.Format)
			if format == "" {
				format = ContentFormatText
			}
			spec.contents = []DocumentContent{contentRecord(tx, content, format)}
		}
		id := createIngestedDocument(tx, spec)
		s.logger.DebugContext(ctx, "uploaded document created",
			slog.String("operation", "document_upload"),
			slog.String("document_id", id),
		)
		return documentDetail(tx, id), nil
	})
}

type ingestedDocumentSpec struct {
	kind      string
	title     string
	url       *string
	sourceURL *string
	folderID  *string
	tagIDs    []string
	contents  []DocumentContent
	blobs     []BlobMetadata
	blobData  map[string][]byte
}

func ingestSpec(tx RepositoryTx, kind, url, sourceURL, title, folderID string, tagIDs []string) (ingestedDocumentSpec, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return ingestedDocumentSpec{}, validation("title is required")
	}
	folder, tags, err := ingestRelationships(tx, folderID, tagIDs)
	if err != nil {
		return ingestedDocumentSpec{}, err
	}
	return ingestedDocumentSpec{
		kind: kind, title: title, url: optionalString(url), sourceURL: optionalString(sourceURL),
		folderID: folder, tagIDs: tags, blobData: map[string][]byte{},
	}, nil
}

func ingestRelationships(tx RepositoryTx, folderID string, tagIDs []string) (*string, []string, error) {
	folder, err := normalizeDocumentFolderID(tx, folderID)
	if err != nil {
		return nil, nil, err
	}
	ids, err := normalizeTagIDs(tx, tagIDs)
	if err != nil {
		return nil, nil, err
	}
	return folder, ids, nil
}

func createIngestedDocument(tx RepositoryTx, spec ingestedDocumentSpec) string {
	now := tx.Now()
	id := tx.NextID("doc")
	tx.PutDocument(documentRecord{
		meta: DocumentMetadata{
			ID: id, Title: spec.title, Kind: spec.kind, FolderID: spec.folderID,
			CanonicalURL: spec.url, SourceURL: spec.sourceURL, Status: DocumentStatusUnread,
			CreatedAt: now, UpdatedAt: now,
		},
		contents: spec.contents,
		blobs:    spec.blobs,
		blobData: spec.blobData,
		tagIDs:   spec.tagIDs,
	})
	return id
}

func contentRecord(tx RepositoryTx, content, format string) DocumentContent {
	return DocumentContent{
		ID: tx.NextID("content"), Role: ContentRoleExtracted,
		Format: format, Content: content,
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
