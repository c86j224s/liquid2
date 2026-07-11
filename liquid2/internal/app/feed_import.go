package app

import (
	"context"
	"log/slog"
	"strings"
)

func (s *Service) ImportFeedItems(ctx context.Context, feedID string, items []FeedImportItem) (FeedImportResult, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (FeedImportResult, error) {
		feed, ok := tx.Feed(feedID)
		if !ok {
			return FeedImportResult{}, notFound("feed")
		}
		if !feed.Enabled {
			return FeedImportResult{}, conflict("feed is disabled")
		}
		result := FeedImportResult{FeedID: feedID, CheckedAt: tx.Now()}
		seen := newFeedItemSet(tx.FeedItems(feedID))
		orphanDocuments := newOrphanFeedDocumentIndex(tx)
		adopted := 0
		for _, item := range items {
			spec, ok := normalizeFeedImportItem(item)
			if !ok || seen.has(spec) {
				result.Skipped++
				continue
			}
			documentID, ok := orphanDocuments.adopt(tx, feed, spec)
			if ok {
				adopted++
			} else {
				var err error
				documentID, err = createFeedItemDocument(tx, feed, spec)
				if err != nil {
					return FeedImportResult{}, err
				}
			}
			tx.PutFeedItem(feedItemRecord(tx, feedID, documentID, spec))
			seen.add(spec)
			result.Imported++
		}
		feed.LastCheckedAt = &result.CheckedAt
		feed.UpdatedAt = result.CheckedAt
		tx.PutFeed(feed)
		s.logger.DebugContext(ctx, "feed items imported",
			slog.String("operation", "feed_import"),
			slog.String("feed_id", feedID),
			slog.Int("imported", result.Imported),
			slog.Int("adopted", adopted),
			slog.Int("skipped", result.Skipped),
		)
		return result, nil
	})
}

type normalizedFeedItem struct {
	title        string
	url          string
	canonicalURL *string
	sourceURL    *string
	guid         *string
	contentHash  *string
	publishedAt  *int64
	content      string
	format       string
}

func normalizeFeedImportItem(item FeedImportItem) (normalizedFeedItem, bool) {
	title := strings.TrimSpace(item.Title)
	url := strings.TrimSpace(item.URL)
	if title == "" || url == "" {
		return normalizedFeedItem{}, false
	}
	format := strings.TrimSpace(item.Format)
	if format == "" {
		format = ContentFormatText
	}
	canonicalURL := optionalString(item.CanonicalURL)
	if canonicalURL == nil {
		canonicalURL = optionalString(url)
	}
	sourceURL := optionalString(item.SourceURL)
	if sourceURL == nil {
		sourceURL = optionalString(url)
	}
	return normalizedFeedItem{
		title:        title,
		url:          url,
		canonicalURL: canonicalURL,
		sourceURL:    sourceURL,
		guid:         optionalString(item.GUID),
		contentHash:  optionalString(item.ContentHash),
		publishedAt:  cloneInt64(item.PublishedAt),
		content:      strings.TrimSpace(item.Content),
		format:       format,
	}, true
}

func createFeedItemDocument(tx RepositoryTx, feed Feed, item normalizedFeedItem) (string, error) {
	folderInput := ""
	if feed.FolderID != nil {
		folderInput = *feed.FolderID
	}
	folderID, err := normalizeFeedDocumentFolderID(tx, folderInput)
	if err != nil {
		return "", err
	}
	spec := ingestedDocumentSpec{
		kind:      DocumentKindRSSItem,
		title:     item.title,
		url:       item.canonicalURL,
		sourceURL: item.sourceURL,
		folderID:  folderID,
		blobData:  map[string][]byte{},
	}
	if item.content != "" {
		spec.contents = []DocumentContent{contentRecord(tx, item.content, item.format)}
	}
	return createIngestedDocument(tx, spec), nil
}

func feedItemRecord(tx RepositoryTx, feedID string, documentID string, item normalizedFeedItem) FeedItem {
	return FeedItem{
		ID: tx.NextID("feed_item"), FeedID: feedID, DocumentID: documentID,
		GUID: item.guid, URL: item.url, CanonicalURL: item.canonicalURL,
		ContentHash: item.contentHash, PublishedAt: item.publishedAt,
		CreatedAt: tx.Now(),
	}
}
