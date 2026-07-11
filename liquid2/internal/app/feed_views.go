package app

import (
	"sort"
	"strings"
)

func cloneFeed(feed Feed) Feed {
	feed.Title = cloneString(feed.Title)
	feed.FolderID = cloneString(feed.FolderID)
	feed.LastCheckedAt = cloneInt64(feed.LastCheckedAt)
	return feed
}

func cloneFeedItem(item FeedItem) FeedItem {
	item.GUID = cloneString(item.GUID)
	item.CanonicalURL = cloneString(item.CanonicalURL)
	item.ContentHash = cloneString(item.ContentHash)
	item.PublishedAt = cloneInt64(item.PublishedAt)
	return item
}

func sortFeeds(feeds []Feed) {
	sort.Slice(feeds, func(i int, j int) bool {
		if feeds[i].CreatedAt != feeds[j].CreatedAt {
			return feeds[i].CreatedAt > feeds[j].CreatedAt
		}
		return feeds[i].ID > feeds[j].ID
	})
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
