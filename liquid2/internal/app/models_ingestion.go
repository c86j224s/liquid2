package app

// BookmarkDocumentInput contains URL bookmark creation fields.
type BookmarkDocumentInput struct {
	// URL is the normalized URL to store.
	URL string
	// SourceURL is the user-provided URL before normalization.
	SourceURL string
	// Title is the optional user-visible document title.
	Title string
	// FolderID categorizes the created document, defaulting to Inbox when empty.
	FolderID string
	// TagIDs assigns tags to the created document.
	TagIDs []string
}

// ScrapedDocumentInput contains extracted URL document fields.
type ScrapedDocumentInput struct {
	// URL is the normalized URL to store.
	URL string
	// SourceURL is the user-provided URL before normalization.
	SourceURL string
	// Title is the extracted or user-visible title.
	Title string
	// Content is the extracted text body.
	Content string
	// Format identifies the extracted body format.
	Format string
	// FolderID categorizes the created document, defaulting to Inbox when empty.
	FolderID string
	// TagIDs assigns tags to the created document.
	TagIDs []string
}

// RescrapeTarget contains the approved source for refreshing a document.
type RescrapeTarget struct {
	// Document is the target document metadata.
	Document DocumentMetadata
	// URL is the source URL to fetch again.
	URL string
}

// RescrapedContentInput contains refreshed scrape content for an existing document.
type RescrapedContentInput struct {
	// URL is the final normalized URL after fetching and redirects.
	URL string
	// SourceURL is the URL used to start the refresh.
	SourceURL string
	// Content is the newly extracted text body.
	Content string
	// Format identifies the extracted body format.
	Format string
}

// UploadedDocumentInput contains file upload document fields.
type UploadedDocumentInput struct {
	// Title is the optional user-visible document title.
	Title string
	// Filename is the uploaded file name.
	Filename string
	// MimeType is the allowlisted upload media type.
	MimeType string
	// Data stores the uploaded bytes.
	Data []byte
	// Content is extracted text when available for this file type.
	Content string
	// Format identifies extracted content format.
	Format string
	// FolderID categorizes the created document, defaulting to Inbox when empty.
	FolderID string
	// TagIDs assigns tags to the created document.
	TagIDs []string
}
