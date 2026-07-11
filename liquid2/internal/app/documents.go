package app

import (
	"context"
	"log/slog"
	"strings"
)

const (
	DocumentKindBookmark       = "bookmark"
	DocumentKindScrapedArticle = "scraped_article"
	DocumentKindUploadedFile   = "uploaded_file"
	DocumentKindRSSItem        = "rss_item"
	DocumentStatusUnread       = "unread"
	DocumentStatusRead         = "read"
)

// documentRecord stores one document and its in-memory relationships.
type documentRecord struct {
	// meta is the document metadata.
	meta DocumentMetadata
	// contents are text bodies attached to the document.
	contents []DocumentContent
	// blobs are binary payloads attached to the document.
	blobs []BlobMetadata
	// blobData stores raw bytes by blob ID for later download support.
	blobData map[string][]byte
	// tagIDs are assigned tag identifiers.
	tagIDs []string
}

func (s *Service) CreateDocument(ctx context.Context, input CreateDocumentInput) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		title := strings.TrimSpace(input.Title)
		if title == "" {
			return DocumentDetail{}, validation("title is required")
		}
		kind := input.Kind
		if kind == "" {
			kind = DocumentKindBookmark
		}
		folderID := ensureDefaultDocumentFolder(tx)
		now := tx.Now()
		id := tx.NextID("doc")
		tx.PutDocument(documentRecord{
			meta: DocumentMetadata{
				ID: id, Title: title, Kind: kind, FolderID: &folderID,
				Status:    DocumentStatusUnread,
				CreatedAt: now, UpdatedAt: now,
			},
			contents: []DocumentContent{},
			blobs:    []BlobMetadata{},
			blobData: map[string][]byte{},
			tagIDs:   []string{},
		})
		s.logger.DebugContext(ctx, "document created", slog.String("operation", "document_create"), slog.String("document_id", id))
		return documentDetail(tx, id), nil
	})
}

func (s *Service) GetDocument(ctx context.Context, id string) (DocumentDetail, error) {
	return withView(ctx, s, func(tx RepositoryReader) (DocumentDetail, error) {
		if _, ok := tx.Document(id); !ok {
			return DocumentDetail{}, notFound("document")
		}
		return documentDetail(tx, id), nil
	})
}

func (s *Service) UpdateDocument(ctx context.Context, id string, input UpdateDocumentInput) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		doc, ok := tx.Document(id)
		if !ok {
			return DocumentDetail{}, notFound("document")
		}
		before := cloneDocumentRecord(doc)
		titleChanged := false
		if input.Title != nil {
			title := strings.TrimSpace(*input.Title)
			if title == "" {
				return DocumentDetail{}, validation("title is required")
			}
			titleChanged = title != doc.meta.Title
			doc.meta.Title = title
		}
		if input.FolderID != nil {
			folderID, err := normalizeDocumentFolderID(tx, *input.FolderID)
			if err != nil {
				return DocumentDetail{}, err
			}
			doc.meta.FolderID = folderID
		}
		now := tx.Now()
		if titleChanged {
			recordDocumentVersion(tx, before, DocumentMutationTitle, now)
		}
		doc.meta.UpdatedAt = now
		tx.PutDocument(doc)
		s.logger.DebugContext(ctx, "document updated", slog.String("operation", "document_update"), slog.String("document_id", id))
		return documentDetail(tx, id), nil
	})
}

func (s *Service) DeleteDocument(ctx context.Context, id string) (int64, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (int64, error) {
		doc, ok := tx.Document(id)
		if !ok {
			return 0, notFound("document")
		}
		now := tx.Now()
		doc.meta.DeletedAt = &now
		doc.meta.UpdatedAt = now
		tx.PutDocument(doc)
		s.logger.DebugContext(ctx, "document soft deleted", slog.String("operation", "document_delete"), slog.String("document_id", id))
		return now, nil
	})
}

func (s *Service) MarkDocumentRead(ctx context.Context, id string) (DocumentDetail, error) {
	return s.setDocumentReadState(ctx, id, true)
}

func (s *Service) MarkDocumentUnread(ctx context.Context, id string) (DocumentDetail, error) {
	return s.setDocumentReadState(ctx, id, false)
}

func (s *Service) SetDocumentRating(ctx context.Context, id string, rating *int) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		doc, ok := tx.Document(id)
		if !ok {
			return DocumentDetail{}, notFound("document")
		}
		if rating != nil && (*rating < 1 || *rating > 5) {
			return DocumentDetail{}, validation("rating must be between 1 and 5")
		}
		doc.meta.Rating = cloneInt(rating)
		doc.meta.UpdatedAt = tx.Now()
		tx.PutDocument(doc)
		s.logger.DebugContext(ctx, "document rating updated", slog.String("operation", "document_rating"), slog.String("document_id", id))
		return documentDetail(tx, id), nil
	})
}

func (s *Service) ReplaceDocumentTags(ctx context.Context, id string, tagIDs []string) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		doc, ok := tx.Document(id)
		if !ok {
			return DocumentDetail{}, notFound("document")
		}
		tagIDs, err := normalizeTagIDs(tx, tagIDs)
		if err != nil {
			return DocumentDetail{}, err
		}
		removedTagIDs := removedDocumentTagIDs(doc.tagIDs, tagIDs)
		doc.tagIDs = tagIDs
		doc.meta.UpdatedAt = tx.Now()
		tx.PutDocument(doc)
		pruned := deleteUnusedTags(tx, removedTagIDs)
		s.logger.DebugContext(ctx, "document tags replaced",
			slog.String("operation", "document_tags"),
			slog.String("document_id", id),
			slog.Int("orphan_tags_deleted", pruned),
		)
		return documentDetail(tx, id), nil
	})
}

func (s *Service) setDocumentReadState(ctx context.Context, id string, read bool) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		doc, ok := tx.Document(id)
		if !ok {
			return DocumentDetail{}, notFound("document")
		}
		now := tx.Now()
		doc.meta.UpdatedAt = now
		if read {
			doc.meta.Status = DocumentStatusRead
			doc.meta.ReadAt = &now
		} else {
			doc.meta.Status = DocumentStatusUnread
			doc.meta.ReadAt = nil
		}
		tx.PutDocument(doc)
		s.logger.DebugContext(ctx, "document read state updated", slog.String("operation", "document_read_state"), slog.String("document_id", id))
		return documentDetail(tx, id), nil
	})
}
