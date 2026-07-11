package app

// Feed is an RSS subscription.
type Feed struct {
	// ID is the stable feed identifier.
	ID string `json:"id"`
	// URL is the RSS or Atom feed URL.
	URL string `json:"url"`
	// Title is the optional user-visible feed title.
	Title *string `json:"title"`
	// FolderID points to the folder used for imported feed documents.
	FolderID *string `json:"folderId"`
	// Enabled controls scheduled polling.
	Enabled bool `json:"enabled"`
	// LastCheckedAt is set after a feed refresh attempt.
	LastCheckedAt *int64 `json:"lastCheckedAt"`
	// CreatedAt is the creation timestamp in Unix milliseconds.
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt is the last update timestamp in Unix milliseconds.
	UpdatedAt int64 `json:"updatedAt"`
}

// FeedItem records one imported RSS item for de-duplication.
type FeedItem struct {
	// ID is the stable feed item identifier.
	ID string `json:"id"`
	// FeedID points to the source feed.
	FeedID string `json:"feedId"`
	// DocumentID points to the imported document.
	DocumentID string `json:"documentId"`
	// GUID is the feed item GUID when provided.
	GUID *string `json:"guid"`
	// URL is the item URL.
	URL string `json:"url"`
	// CanonicalURL is a normalized item URL when known.
	CanonicalURL *string `json:"canonicalUrl"`
	// ContentHash identifies items without stable GUID or URL.
	ContentHash *string `json:"contentHash"`
	// PublishedAt is the feed item publication timestamp.
	PublishedAt *int64 `json:"publishedAt"`
	// CreatedAt is the import timestamp in Unix milliseconds.
	CreatedAt int64 `json:"createdAt"`
}

// FeedImportItem contains one normalized external feed entry ready for import.
type FeedImportItem struct {
	// Title is the user-visible document title.
	Title string
	// URL is the item URL.
	URL string
	// CanonicalURL is the normalized item URL when known.
	CanonicalURL string
	// SourceURL is the original item URL from the feed.
	SourceURL string
	// GUID is the feed item GUID when provided.
	GUID string
	// ContentHash identifies items without stable GUID or URL.
	ContentHash string
	// PublishedAt is the item publication timestamp.
	PublishedAt *int64
	// Content is the extracted feed item body when available.
	Content string
	// Format identifies the body format.
	Format string
}

// FeedImportResult summarizes an import attempt.
type FeedImportResult struct {
	// FeedID is the source feed.
	FeedID string
	// Imported counts newly accepted feed items.
	Imported int
	// Skipped counts duplicate or unusable feed items.
	Skipped int
	// CheckedAt is the feed refresh timestamp.
	CheckedAt int64
}

// CreateFeedInput contains fields accepted when creating a feed.
type CreateFeedInput struct {
	// URL is the RSS or Atom feed URL.
	URL string
	// Title is the optional user-visible title.
	Title *string
	// FolderID is rejected on create; feed folders are assigned automatically.
	FolderID *string
	// Enabled overrides the default enabled state when set.
	Enabled *bool
}

// UpdateFeedInput contains mutable feed fields.
type UpdateFeedInput struct {
	// URL replaces the feed URL when set.
	URL *string
	// Title replaces or clears the title when set.
	Title *string
	// FolderID replaces or clears the target folder when set.
	FolderID *string
	// Enabled replaces the polling enabled flag when set.
	Enabled *bool
}
