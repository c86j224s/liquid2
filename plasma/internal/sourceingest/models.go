package sourceingest

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type Store interface {
	ListSourceSnapshots(context.Context, string) ([]app.SourceSnapshot, error)
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
	GetRawArtifact(context.Context, string) (app.RawArtifact, error)
	CreateSourceSnapshotWithEvent(context.Context, app.CreateSourceSnapshotWithEventRequest) (app.SourceSnapshotWithEventResult, error)
	CreateExistingArtifactSourceSnapshotWithEvent(context.Context, app.CreateExistingArtifactSourceSnapshotWithEventRequest) (app.ExistingArtifactSourceSnapshotWithEventResult, error)
	CreateLiveSourceSnapshotWithEvent(context.Context, app.CreateLiveSourceSnapshotWithEventRequest) (app.LiveSourceSnapshotWithEventResult, error)
	AppendEvent(context.Context, app.AppendEventRequest) (app.LedgerEvent, error)
}

type Producer = app.Producer
type RawArtifact = app.RawArtifact
type SourceSnapshot = app.SourceSnapshot
type LedgerEvent = app.LedgerEvent
type SourceSnapshotWithEventResult = app.SourceSnapshotWithEventResult
type ExistingArtifactSourceSnapshotWithEventResult = app.ExistingArtifactSourceSnapshotWithEventResult
type LiveSourceSnapshotWithEventResult = app.LiveSourceSnapshotWithEventResult
type AppendEventRequest = app.AppendEventRequest
type CreateRawArtifactRequest = app.CreateRawArtifactRequest
type CreateSourceSnapshotRequest = app.CreateSourceSnapshotRequest
type CreateSourceSnapshotWithEventRequest = app.CreateSourceSnapshotWithEventRequest
type CreateExistingArtifactSourceSnapshotWithEventRequest = app.CreateExistingArtifactSourceSnapshotWithEventRequest
type CreateLiveSourceSnapshotWithEventRequest = app.CreateLiveSourceSnapshotWithEventRequest
type ConnectorRef = app.ConnectorRef
type MediaLocator = app.MediaLocator
type SourceAccess = app.SourceAccess

const SourceLocatorTypeFullDocument = app.SourceLocatorTypeFullDocument
const SourceLocatorTypePDFDocument = app.SourceLocatorTypePDFDocument
const SourceLocatorTypeMedia = app.SourceLocatorTypeMedia
const SourceConnectorTypePDFURL = app.SourceConnectorTypePDFURL
const SourceConnectorTypeMediaURL = app.SourceConnectorTypeMediaURL
const MediaKindImage = app.MediaKindImage
const MediaKindAudio = app.MediaKindAudio
const MediaKindVideo = app.MediaKindVideo
const SourceRetrievalPolicyLiveReference = app.SourceRetrievalPolicyLiveReference

var ErrInvalidInput = app.ErrInvalidInput

type FetchedURLSource struct {
	Content           []byte
	MediaType         string
	Title             string
	ExternalVersion   string
	ExternalUpdatedAt time.Time
	ByteSize          int64
	PageCount         int
	TextLength        int
	TextLengthKnown   bool
}

type FetchedMediaSource struct {
	Content           []byte
	MediaType         string
	MediaKind         string
	Title             string
	ExternalVersion   string
	ExternalUpdatedAt time.Time
	ByteSize          int64
	Width             int
	Height            int
}

type TextSourceContent struct {
	Title       string
	Content     string
	ExternalURI string
}

type StagedSourceCandidate struct {
	URL               string
	Title             string
	ProposalEventID   string
	Artifact          RawArtifact
	ExternalVersion   string
	ExternalUpdatedAt time.Time
}

type URLSourceSnapshotResult struct {
	Artifact              RawArtifact
	Snapshot              SourceSnapshot
	Event                 LedgerEvent
	ReusedSourceCandidate bool
}

type MediaSourceSnapshotResult struct {
	Artifact    RawArtifact
	HasArtifact bool
	Snapshot    SourceSnapshot
	Event       LedgerEvent
}

type CreateFetchedURLSourceRequest struct {
	MissionID  string
	URL        string
	Title      string
	ArtifactID string
	SnapshotID string
	EventID    string
	Producer   Producer
	Fetched    FetchedURLSource
	FetchedAt  time.Time
}

type SourceSnapshotFailureAppendRequest struct {
	EventID    string
	MissionID  string
	SourceKind string
	URL        string
	Message    string
	Producer   Producer
}

type CreateTextSourceWithEventRequest struct {
	MissionID  string
	ArtifactID string
	SnapshotID string
	EventID    string
	Producer   Producer
	Source     TextSourceContent
}

type CreateStagedURLSourceRequest struct {
	MissionID  string
	URL        string
	Title      string
	SnapshotID string
	EventID    string
	Producer   Producer
	Staged     StagedSourceCandidate
}

type CreateFetchedPDFURLSourceRequest struct {
	MissionID  string
	URL        string
	Title      string
	ArtifactID string
	SnapshotID string
	EventID    string
	Producer   Producer
	Fetched    FetchedURLSource
	FetchedAt  time.Time
}

type CreateFetchedMediaURLSourceRequest struct {
	MissionID   string
	URL         string
	Title       string
	License     string
	Attribution string
	ArtifactID  string
	SnapshotID  string
	EventID     string
	Producer    Producer
	Fetched     FetchedMediaSource
}

type CreateStagedPDFURLSourceRequest struct {
	MissionID  string
	URL        string
	Title      string
	SnapshotID string
	EventID    string
	Producer   Producer
	Staged     StagedSourceCandidate
}
