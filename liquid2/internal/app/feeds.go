package app

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
)

func (s *Service) ListFeeds(ctx context.Context) ([]Feed, error) {
	return withView(ctx, s, func(tx RepositoryReader) ([]Feed, error) {
		feeds := tx.Feeds()
		sortFeeds(feeds)
		return feeds, nil
	})
}

func (s *Service) GetFeed(ctx context.Context, id string) (Feed, error) {
	return withView(ctx, s, func(tx RepositoryReader) (Feed, error) {
		feed, ok := tx.Feed(id)
		if !ok {
			return Feed{}, notFound("feed")
		}
		return feed, nil
	})
}

func (s *Service) CreateFeed(ctx context.Context, input CreateFeedInput) (Feed, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (Feed, error) {
		urlValue, err := validateFeedURL(input.URL)
		if err != nil {
			return Feed{}, err
		}
		if _, ok := tx.FeedByURL(urlValue); ok {
			return Feed{}, conflict("feed url already exists")
		}
		if input.FolderID != nil && strings.TrimSpace(*input.FolderID) != "" {
			return Feed{}, validation("feed folder is assigned automatically")
		}
		now := tx.Now()
		enabled := true
		if input.Enabled != nil {
			enabled = *input.Enabled
		}
		title := normalizeOptionalText(input.Title)
		folderID := createFeedDocumentFolder(tx, title, urlValue)
		feed := Feed{
			ID: tx.NextID("feed"), URL: urlValue, Title: title,
			FolderID: &folderID, Enabled: enabled, CreatedAt: now, UpdatedAt: now,
		}
		tx.PutFeed(feed)
		s.logger.DebugContext(ctx, "feed created", slog.String("operation", "feed_create"), slog.String("feed_id", feed.ID))
		return feed, nil
	})
}

func (s *Service) UpdateFeed(ctx context.Context, id string, input UpdateFeedInput) (Feed, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (Feed, error) {
		feed, ok := tx.Feed(id)
		if !ok {
			return Feed{}, notFound("feed")
		}
		if input.URL != nil {
			urlValue, err := validateFeedURL(*input.URL)
			if err != nil {
				return Feed{}, err
			}
			if existing, ok := tx.FeedByURL(urlValue); ok && existing.ID != id {
				return Feed{}, conflict("feed url already exists")
			}
			feed.URL = urlValue
		}
		if input.Title != nil {
			feed.Title = normalizeOptionalText(input.Title)
		}
		if input.FolderID != nil {
			folderID, err := normalizeFeedFolder(tx, input.FolderID)
			if err != nil {
				return Feed{}, err
			}
			feed.FolderID = folderID
		}
		if input.Enabled != nil {
			feed.Enabled = *input.Enabled
		}
		feed.UpdatedAt = tx.Now()
		tx.PutFeed(feed)
		s.logger.DebugContext(ctx, "feed updated", slog.String("operation", "feed_update"), slog.String("feed_id", feed.ID))
		return feed, nil
	})
}

func (s *Service) DeleteFeed(ctx context.Context, id string) error {
	_, err := withUpdate(ctx, s, func(tx RepositoryTx) (struct{}, error) {
		if _, ok := tx.Feed(id); !ok {
			return struct{}{}, notFound("feed")
		}
		tx.DeleteFeed(id)
		s.logger.DebugContext(ctx, "feed deleted", slog.String("operation", "feed_delete"), slog.String("feed_id", id))
		return struct{}{}, nil
	})
	return err
}

func validateFeedURL(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", validation("feed url is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return "", validation("feed url is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", validation("feed url must use http or https")
	}
	return trimmed, nil
}

func normalizeFeedFolder(tx RepositoryReader, value *string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	folderID := strings.TrimSpace(*value)
	if _, ok := tx.Folder(folderID); !ok {
		return nil, notFound("folder")
	}
	if folderHasSystemRoleAncestor(tx, folderID, FolderSystemRoleTrash) {
		return nil, validation("folder cannot be trash")
	}
	return &folderID, nil
}
