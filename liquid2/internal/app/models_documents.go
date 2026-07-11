package app

const (
	ContentRoleOriginal    = "original"
	ContentRoleExtracted   = "extracted"
	ContentRoleTranslation = "translation"
	ContentRoleSummary     = "summary"
	ContentFormatHTML      = "html"
	ContentFormatMarkdown  = "markdown"
	ContentFormatText      = "text"
)

// DocumentMetadata is the mutable metadata stored for a document.
type DocumentMetadata struct {
	// ID is the stable document identifier.
	ID string `json:"id"`
	// Title is the user-visible document title.
	Title string `json:"title"`
	// Kind identifies the document source or creation workflow.
	Kind string `json:"kind"`
	// FolderID points to the containing folder.
	FolderID *string `json:"folderId"`
	// CanonicalURL is the normalized source URL when known.
	CanonicalURL *string `json:"canonicalUrl"`
	// SourceURL is the original URL captured from the user or feed.
	SourceURL *string `json:"sourceUrl"`
	// Language is the detected or user-assigned content language.
	Language *string `json:"language"`
	// Status is the read-state status such as unread or read.
	Status string `json:"status"`
	// Rating is the optional user rating from 1 to 5.
	Rating *int `json:"rating"`
	// CreatedAt is the creation timestamp in Unix milliseconds.
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt is the last metadata update timestamp in Unix milliseconds.
	UpdatedAt int64 `json:"updatedAt"`
	// PublishedAt is the original source publication timestamp when known.
	PublishedAt *int64 `json:"publishedAt"`
	// ReadAt is set when the document was last marked read.
	ReadAt *int64 `json:"readAt"`
	// DeletedAt is set when the document has been soft-deleted.
	DeletedAt *int64 `json:"deletedAt"`
}

// DocumentSummary is the compact document shape used in list responses.
type DocumentSummary struct {
	// ID is the stable document identifier.
	ID string `json:"id"`
	// Title is the user-visible document title.
	Title string `json:"title"`
	// Kind identifies the document source or creation workflow.
	Kind string `json:"kind"`
	// FolderID points to the containing folder.
	FolderID *string `json:"folderId"`
	// FolderPath contains display breadcrumbs from root folder to document folder.
	FolderPath []FolderBreadcrumb `json:"folderPath"`
	// CanonicalURL is the normalized source URL when known.
	CanonicalURL *string `json:"canonicalUrl"`
	// SourceURL is the original URL captured from the user or feed.
	SourceURL *string `json:"sourceUrl"`
	// Language is the detected or user-assigned content language.
	Language *string `json:"language"`
	// Status is the read-state status such as unread or read.
	Status string `json:"status"`
	// Rating is the optional user rating from 1 to 5.
	Rating *int `json:"rating"`
	// CreatedAt is the creation timestamp in Unix milliseconds.
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt is the last metadata update timestamp in Unix milliseconds.
	UpdatedAt int64 `json:"updatedAt"`
	// PublishedAt is the original source publication timestamp when known.
	PublishedAt *int64 `json:"publishedAt"`
	// ReadAt is set when the document was last marked read.
	ReadAt *int64 `json:"readAt"`
	// DeletedAt is set when the document has been soft-deleted.
	DeletedAt *int64 `json:"deletedAt"`
	// Tags contains the document tag slugs for quick display and filtering.
	Tags []string `json:"tags"`
}

// DocumentContent is a content body associated with a document.
type DocumentContent struct {
	// ID is the stable content block identifier.
	ID string `json:"id"`
	// Role describes the content purpose, such as original or translated.
	Role string `json:"role"`
	// Format identifies the body format, such as text or markdown.
	Format string `json:"format"`
	// Language is the content language when known.
	Language *string `json:"language"`
	// Content is the stored text body.
	Content string `json:"content"`
	// SourceContentID links derived variants to their source content.
	SourceContentID *string `json:"-"`
}

// BlobMetadata describes an uploaded or captured binary payload.
type BlobMetadata struct {
	// ID is the stable blob identifier.
	ID string `json:"id"`
	// Filename is the original or assigned file name.
	Filename string `json:"filename"`
	// MimeType is the payload media type.
	MimeType string `json:"mimeType"`
	// Size is the payload size in bytes.
	Size int64 `json:"size"`
	// CreatedAt is the creation timestamp in Unix milliseconds.
	CreatedAt int64 `json:"createdAt"`
}

// DocumentDetail is the complete document response shape.
type DocumentDetail struct {
	// Document contains document metadata.
	Document DocumentMetadata `json:"document"`
	// FolderPath contains display breadcrumbs from root folder to document folder.
	FolderPath []FolderBreadcrumb `json:"folderPath"`
	// Contents contains the document text bodies.
	Contents []DocumentContent `json:"contents"`
	// Tags contains full tag records assigned to the document.
	Tags []Tag `json:"tags"`
	// Blobs contains file payload metadata attached to the document.
	Blobs []BlobMetadata `json:"blobs"`
}

// DocumentList is a paginated document list response.
type DocumentList struct {
	// Items contains the current page of document summaries.
	Items []DocumentSummary `json:"items"`
	// NextCursor is set when another page is available.
	NextCursor *string `json:"nextCursor"`
	// TotalCount is exact on the first page and -1 when skipped for cursor pages.
	TotalCount int `json:"totalCount"`
}

// DocumentFilters contains document list filter criteria.
type DocumentFilters struct {
	// Query filters documents by plain text search query.
	Query string
	// Status filters documents by read-state status.
	Status string
	// FolderID filters documents to a specific folder.
	FolderID string
	// IncludeFolderDescendants includes child folders when FolderID is set.
	IncludeFolderDescendants bool
	// Tag filters documents by tag slug.
	Tag string
	// RatingMin filters documents with rating greater than or equal to this value.
	RatingMin int
	// Kind filters documents by document kind.
	Kind string
	// Sort controls document list ordering.
	Sort string
	// IncludeDeleted includes soft-deleted documents when true.
	IncludeDeleted bool
	// IncludeTrash includes documents assigned to the Trash system folder.
	IncludeTrash bool
	// Limit caps the number of returned documents.
	Limit int
	// Cursor identifies the next page to fetch.
	Cursor string
}

// CreateDocumentInput contains the fields accepted when creating a document.
type CreateDocumentInput struct {
	// Title is the user-visible document title.
	Title string
	// Kind identifies the document source or creation workflow.
	Kind string
}

// UpdateDocumentInput contains mutable document fields.
type UpdateDocumentInput struct {
	// Title replaces the document title when set.
	Title *string
	// FolderID moves the document to a folder, or to the default folder when empty.
	FolderID *string
}

// AppendTranslationInput contains a completed translation variant.
type AppendTranslationInput struct {
	// SourceContentID identifies the source content variant.
	SourceContentID string
	// TargetLanguage is the translated content language tag.
	TargetLanguage string
	// Content is the translated text body.
	Content string
	// Format identifies the translated body format.
	Format string
}

// PrepareTranslationInput contains source fields needed before provider work.
type PrepareTranslationInput struct {
	// SourceContentID identifies the source content variant.
	SourceContentID string
	// TargetLanguage is the translated content language tag.
	TargetLanguage string
}
