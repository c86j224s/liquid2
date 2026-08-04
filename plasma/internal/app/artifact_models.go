package app

import (
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

const (
	SourceRetrievalPolicySnapshotOnly  = source.RetrievalPolicySnapshotOnly
	SourceRetrievalPolicyLiveReference = source.RetrievalPolicyLiveReference

	SourceConnectorTypeLocalPath  = source.ConnectorTypeLocalPath
	SourceConnectorTypeMediaURL   = source.ConnectorTypeMediaURL
	SourceConnectorTypePDFURL     = source.ConnectorTypePDFURL
	SourceConnectorTypeFileUpload = source.ConnectorTypeFileUpload

	SourceLocatorTypeFullText     = source.LocatorTypeFullText
	SourceLocatorTypeFullDocument = source.LocatorTypeFullDocument
	SourceLocatorTypePDFDocument  = source.LocatorTypePDFDocument
	SourceLocatorTypeMedia        = source.LocatorTypeMedia
	SourceLocatorTypeLocalPath    = source.LocatorTypeLocalPath

	MediaKindImage = source.MediaKindImage
	MediaKindAudio = source.MediaKindAudio
	MediaKindVideo = source.MediaKindVideo

	SourceStateActive  = source.StateActive
	SourceStateRemoved = source.StateRemoved
)

// Deprecated: capability code should import internal/artifact directly.
type RawArtifact = artifact.Raw
type CreateRawArtifactRequest = artifact.CreateRequest

// Deprecated: capability code should import internal/source directly.
type ConnectorRef = source.ConnectorRef
type ContentHash = source.ContentHash
type SourceAccess = source.Access
type SourceSnapshot = source.Snapshot
type CreateSourceSnapshotRequest = source.CreateRequest
type SourceState = source.State
type LocalPathLocator = source.LocalPathLocator
type MediaLocator = source.MediaLocator
type UploadedFileLocator = source.UploadedFileLocator
type ListSourceSnapshotsRequest = source.ListRequest
