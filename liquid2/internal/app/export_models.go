package app

import "context"

const (
	ExportManifestVersion = 1
	exportAppName         = "liquid2"
)

type MarkdownExportInput struct {
	ExportID      string
	CreatedAt     int64
	AppVersion    *string
	SchemaVersion *int64
	DocumentIDs   []string
}

type MarkdownExportResult struct {
	Manifest      ExportManifest
	DocumentCount int
	BlobCount     int
}

type ExportWriter interface {
	WriteFile(ctx context.Context, path string, data []byte) error
}

type ExportManifest struct {
	ManifestVersion int              `json:"manifestVersion"`
	ExportID        string           `json:"exportId"`
	CreatedAt       int64            `json:"createdAt"`
	Source          ExportSource     `json:"source"`
	Counts          ExportCounts     `json:"counts"`
	Documents       []ExportDocument `json:"documents"`
}

type ExportSource struct {
	App           string  `json:"app"`
	AppVersion    *string `json:"appVersion"`
	SchemaVersion *int64  `json:"schemaVersion"`
}

type ExportCounts struct {
	Documents int `json:"documents"`
	Blobs     int `json:"blobs"`
}

type ExportDocument struct {
	ID           string          `json:"id"`
	MarkdownPath string          `json:"markdownPath"`
	Title        string          `json:"title"`
	Kind         string          `json:"kind"`
	FolderID     *string         `json:"folderId"`
	SourceURL    *string         `json:"sourceUrl"`
	CanonicalURL *string         `json:"canonicalUrl"`
	Language     *string         `json:"language"`
	Contents     []ExportContent `json:"contents"`
	Tags         []ExportTag     `json:"tags"`
	Blobs        []ExportBlob    `json:"blobs"`
}

type ExportContent struct {
	ID              string  `json:"id"`
	Role            string  `json:"role"`
	Format          string  `json:"format"`
	Language        *string `json:"language"`
	SourceContentID *string `json:"sourceContentId,omitempty"`
}

type ExportTag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ExportBlob struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mimeType"`
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
}
