package app

import sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"

func (tx *sqliteTx) Feed(id string) (Feed, bool) {
	row, err := tx.q.GetFeed(tx.ctx, id)
	if tx.missing(err) {
		return Feed{}, false
	}
	return sqliteFeed(row), true
}

func (tx *sqliteTx) FeedByURL(url string) (Feed, bool) {
	row, err := tx.q.GetFeedByURL(tx.ctx, url)
	if tx.missing(err) {
		return Feed{}, false
	}
	return sqliteFeed(row), true
}

func (tx *sqliteTx) Feeds() []Feed {
	rows, err := tx.q.ListFeeds(tx.ctx)
	tx.abort(err)
	feeds := make([]Feed, 0, len(rows))
	for _, row := range rows {
		feeds = append(feeds, sqliteFeed(row))
	}
	return feeds
}

func (tx *sqliteTx) FeedItemByDocumentID(documentID string) (FeedItem, bool) {
	row, err := tx.q.GetFeedItemByDocumentID(tx.ctx, documentID)
	if tx.missing(err) {
		return FeedItem{}, false
	}
	tx.abort(err)
	return sqliteFeedItem(row), true
}

func (tx *sqliteTx) FeedItems(feedID string) []FeedItem {
	rows, err := tx.q.ListFeedItems(tx.ctx, feedID)
	tx.abort(err)
	items := make([]FeedItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqliteFeedItem(row))
	}
	return items
}

func (tx *sqliteTx) PutFeed(feed Feed) {
	_, err := tx.q.UpsertFeed(tx.ctx, sqlitedb.UpsertFeedParams{
		ID: feed.ID, Url: feed.URL, Title: sqliteNullString(feed.Title),
		FolderID: sqliteNullString(feed.FolderID), Enabled: sqliteBool(feed.Enabled),
		LastCheckedAt: sqliteNullInt64(feed.LastCheckedAt), CreatedAt: feed.CreatedAt, UpdatedAt: feed.UpdatedAt,
	})
	tx.abort(err)
}

func (tx *sqliteTx) DeleteFeed(id string) {
	tx.abort(tx.q.DeleteFeed(tx.ctx, id))
}

func (tx *sqliteTx) PutFeedItem(item FeedItem) {
	_, err := tx.q.UpsertFeedItem(tx.ctx, sqlitedb.UpsertFeedItemParams{
		ID: item.ID, FeedID: item.FeedID, DocumentID: item.DocumentID, Guid: sqliteNullString(item.GUID),
		Url: item.URL, CanonicalUrl: sqliteNullString(item.CanonicalURL), ContentHash: sqliteNullString(item.ContentHash),
		PublishedAt: sqliteNullInt64(item.PublishedAt), CreatedAt: item.CreatedAt,
	})
	tx.abort(err)
}
