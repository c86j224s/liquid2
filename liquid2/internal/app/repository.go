package app

import "context"

// Repository stores document-library aggregates behind app service rules.
// Adapters currently live in package app because documentRecord is the app-owned aggregate shape.
type Repository interface {
	View(ctx context.Context, fn func(RepositoryReader) error) error
	Update(ctx context.Context, fn func(RepositoryTx) error) error
	Close() error
}

// RepositoryReader is the read-only app-owned persistence view.
type RepositoryReader interface {
	Document(id string) (documentRecord, bool)
	ListDocumentIDs(filters DocumentFilters) ([]string, *string, error)
	CountDocuments(filters DocumentFilters) (int, error)
	Documents() []documentRecord
	DocumentVersions(documentID string) []DocumentVersion

	Folder(id string) (Folder, bool)
	Folders() []Folder

	Tag(id string) (Tag, bool)
	TagBySlug(slug string) (Tag, bool)
	TagHasDocuments(id string) bool
	Tags() []Tag

	Feed(id string) (Feed, bool)
	FeedByURL(url string) (Feed, bool)
	Feeds() []Feed
	FeedItemByDocumentID(documentID string) (FeedItem, bool)
	FeedItems(feedID string) []FeedItem

	Job(id string) (Job, bool)
	Jobs(filters JobFilters) []Job

	DocumentNotes(documentID string) []DocumentNote
	Note(documentID string, noteID string) (DocumentNote, bool)

	Settings() AppSettings
}

// RepositoryTx is the app-owned writable persistence transaction surface.
// Implementations must make writes visible to later reads in the same callback.
type RepositoryTx interface {
	RepositoryReader

	Now() int64
	NextID(prefix string) string
	PutDocument(record documentRecord)
	PutDocumentVersion(version DocumentVersion)
	PutFolder(folder Folder)
	DeleteFolder(id string)
	PutTag(tag Tag)
	DeleteTag(id string)
	PutFeed(feed Feed)
	DeleteFeed(id string)
	PutFeedItem(item FeedItem)
	PutJob(job Job)
	PutNote(note DocumentNote)
	PutSettings(settings AppSettings)
}
