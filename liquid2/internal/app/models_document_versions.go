package app

const (
	DocumentMutationTitle   = "title"
	DocumentMutationContent = "content"
)

// DocumentVersion stores an old document snapshot captured before mutation.
type DocumentVersion struct {
	// ID is the stable version identifier.
	ID string
	// DocumentID identifies the document whose old state was captured.
	DocumentID string
	// Sequence is a per-document increasing version number.
	Sequence int64
	// MutationKind describes the app rule that created the version.
	MutationKind string
	// Title stores the old document title.
	Title string
	// Contents stores the old content variants.
	Contents []DocumentContent
	// Metadata stores the old document metadata snapshot.
	Metadata DocumentMetadata
	// CreatedAt is the capture timestamp in Unix milliseconds.
	CreatedAt int64
}
