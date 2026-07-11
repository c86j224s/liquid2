package app

import (
	"encoding/json"
	"time"
)

const (
	SourceRetrievalPolicySnapshotOnly  = "snapshot_only"
	SourceRetrievalPolicyLiveReference = "live_reference"

	SourceConnectorTypeLocalPath  = "local_path"
	SourceConnectorTypeMediaURL   = "media_url"
	SourceConnectorTypePDFURL     = "pdf_url"
	SourceConnectorTypeFileUpload = "file_upload"

	SourceLocatorTypeFullText     = "full_text"
	SourceLocatorTypeFullDocument = "full_document"
	SourceLocatorTypePDFDocument  = "pdf_document"
	SourceLocatorTypeMedia        = "media"
	SourceLocatorTypeLocalPath    = SourceConnectorTypeLocalPath

	MediaKindImage = "image"
	MediaKindAudio = "audio"
	MediaKindVideo = "video"

	SourceStateActive  = "active"
	SourceStateRemoved = "removed"
)

type RawArtifact struct {
	ArtifactID string
	MissionID  string
	MediaType  string
	ByteSize   int64
	SHA256     string
	StorageURI string
	Filename   string
	Producer   Producer
	CreatedAt  time.Time
	Content    []byte
}

type ConnectorRef struct {
	ConnectorID      string
	ConnectorType    string
	ExternalSourceID string
	ExternalURI      string
	ExternalVersion  string
	ConnectorVersion string
}

type ContentHash struct {
	Algorithm string
	Value     string
}

type SourceAccess struct {
	Visibility      string
	License         string
	RetrievalPolicy string
}

type SourceSnapshot struct {
	SnapshotID        string
	MissionID         string
	Connector         ConnectorRef
	Title             string
	CapturedAt        time.Time
	ExternalUpdatedAt time.Time
	ArtifactIDs       []string
	ContentHash       ContentHash
	Locators          json.RawMessage
	Access            SourceAccess
	State             SourceState
}

type CreateRawArtifactRequest struct {
	ArtifactID     string
	MissionID      string
	MediaType      string
	Filename       string
	Producer       Producer
	Content        []byte
	ExpectedSHA256 string
}

type CreateSourceSnapshotRequest struct {
	SnapshotID        string
	MissionID         string
	Connector         ConnectorRef
	Title             string
	ExternalUpdatedAt time.Time
	ArtifactIDs       []string
	ContentHash       ContentHash
	Locators          json.RawMessage
	Access            SourceAccess
}

type SourceState struct {
	State             string    `json:"state,omitempty"`
	Removed           bool      `json:"removed,omitempty"`
	RemovedAt         time.Time `json:"removed_at,omitempty"`
	RemovedEventID    string    `json:"removed_event_id,omitempty"`
	RemovedReason     string    `json:"removed_reason,omitempty"`
	RestoredAt        time.Time `json:"restored_at,omitempty"`
	RestoredEventID   string    `json:"restored_event_id,omitempty"`
	Superseded        bool      `json:"superseded,omitempty"`
	SupersededAt      time.Time `json:"superseded_at,omitempty"`
	SupersededBy      string    `json:"superseded_by,omitempty"`
	SupersededEventID string    `json:"superseded_event_id,omitempty"`
}

type LocalPathLocator struct {
	LocatorType  string `json:"locator_type,omitempty"`
	Kind         string `json:"kind,omitempty"` // Legacy discriminator accepted on read.
	RootID       string `json:"root_id"`
	RelativePath string `json:"relative_path"`
	PathKind     string `json:"path_kind"`
}

type MediaLocator struct {
	LocatorType       string `json:"locator_type,omitempty"`
	Kind              string `json:"kind,omitempty"` // Legacy discriminator accepted on read.
	MediaKind         string `json:"media_kind"`
	Provider          string `json:"provider,omitempty"`
	CanonicalURL      string `json:"canonical_url,omitempty"`
	SourcePageURL     string `json:"source_page_url,omitempty"`
	DirectMediaURL    string `json:"direct_media_url,omitempty"`
	MIMEType          string `json:"mime_type,omitempty"`
	ByteSize          int64  `json:"byte_size,omitempty"`
	Width             int    `json:"width,omitempty"`
	Height            int    `json:"height,omitempty"`
	DurationMS        int64  `json:"duration_ms,omitempty"`
	Codec             string `json:"codec,omitempty"`
	Title             string `json:"title,omitempty"`
	Attribution       string `json:"attribution,omitempty"`
	License           string `json:"license,omitempty"`
	SHA256            string `json:"sha256,omitempty"`
	InspectionSupport string `json:"inspection_support,omitempty"`
}

type UploadedFileLocator struct {
	LocatorType       string    `json:"locator_type"`
	Kind              string    `json:"kind,omitempty"` // Legacy file_upload discriminator accepted on read.
	MediaKind         string    `json:"media_kind,omitempty"`
	OriginalFilename  string    `json:"original_filename"`
	SanitizedFilename string    `json:"sanitized_filename"`
	MIMEType          string    `json:"mime_type,omitempty"`
	MediaType         string    `json:"media_type,omitempty"` // Legacy upload locator field accepted on read.
	ByteSize          int64     `json:"byte_size"`
	SHA256            string    `json:"sha256"`
	UploadedAt        time.Time `json:"uploaded_at"`
	ContentKind       string    `json:"content_kind"`
}

type ListSourceSnapshotsRequest struct {
	MissionID         string
	IncludeRemoved    bool
	IncludeSuperseded bool
}
