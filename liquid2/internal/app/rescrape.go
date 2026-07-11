package app

import (
	"context"
	"log/slog"
	"strings"
)

func (s *Service) PrepareRescrape(ctx context.Context, id string) (RescrapeTarget, error) {
	return withView(ctx, s, func(tx RepositoryReader) (RescrapeTarget, error) {
		doc, err := rescrapeDocument(tx, id)
		if err != nil {
			return RescrapeTarget{}, err
		}
		return RescrapeTarget{
			Document: cloneDocumentMetadata(doc.meta),
			URL:      rescrapeURL(doc.meta),
		}, nil
	})
}

func (s *Service) ReplaceRescrapedContent(
	ctx context.Context,
	id string,
	input RescrapedContentInput,
) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		doc, err := rescrapeDocument(tx, id)
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
		if !validContentFormat(format) {
			return DocumentDetail{}, validation("content format is invalid")
		}

		before := cloneDocumentRecord(doc)
		updateExtractedContent(tx, &doc, content, format)
		doc.meta.CanonicalURL = optionalString(input.URL)
		if sourceURL := optionalString(input.SourceURL); sourceURL != nil {
			doc.meta.SourceURL = sourceURL
		}
		now := tx.Now()
		recordDocumentVersion(tx, before, DocumentMutationContent, now)
		doc.meta.UpdatedAt = now
		tx.PutDocument(doc)
		s.logger.DebugContext(ctx, "document re-scraped",
			slog.String("operation", "document_rescrape"),
			slog.String("document_id", id),
		)
		return documentDetail(tx, id), nil
	})
}

func rescrapeDocument(tx RepositoryReader, id string) (documentRecord, error) {
	doc, ok := tx.Document(id)
	if !ok || doc.meta.DeletedAt != nil {
		return documentRecord{}, notFound("document")
	}
	if !rescrapableDocumentKind(doc.meta.Kind) {
		return documentRecord{}, validation("document cannot be re-scraped")
	}
	if rescrapeURL(doc.meta) == "" {
		return documentRecord{}, validation("document has no source URL")
	}
	return doc, nil
}

func rescrapableDocumentKind(kind string) bool {
	return kind == DocumentKindScrapedArticle || kind == DocumentKindRSSItem
}

func rescrapeURL(meta DocumentMetadata) string {
	if meta.SourceURL != nil && strings.TrimSpace(*meta.SourceURL) != "" {
		return strings.TrimSpace(*meta.SourceURL)
	}
	if meta.CanonicalURL != nil && strings.TrimSpace(*meta.CanonicalURL) != "" {
		return strings.TrimSpace(*meta.CanonicalURL)
	}
	return ""
}

func updateExtractedContent(
	tx RepositoryTx,
	doc *documentRecord,
	content string,
	format string,
) {
	for i, current := range doc.contents {
		if current.Role == ContentRoleExtracted {
			doc.contents[i].Format = format
			doc.contents[i].Content = content
			doc.contents[i].SourceContentID = nil
			return
		}
	}
	doc.contents = append(doc.contents, contentRecord(tx, content, format))
}
