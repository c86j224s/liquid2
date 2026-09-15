package liquid2source

import (
	"context"
	"encoding/json"
	"time"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

const (
	// Liquid2Connector* constants define the Liquid2 connector identity and snapshot schema.
	Liquid2ConnectorID       = "liquid2"
	Liquid2ConnectorType     = "liquid2"
	Liquid2HTTPConnectorV1   = "liquid2-http.v1"
	Liquid2SnapshotMediaType = "application/vnd.plasma.liquid2.snapshot+json"
	Liquid2SnapshotSchemaV1  = "plasma.liquid2.snapshot.v1"
	defaultSearchLimit       = 10
	maxSearchLimit           = 100
)

// Liquid2SourceConnector is the consumer-side port for reading Liquid2 documents.
type Liquid2SourceConnector interface {
	SearchLiquid2Sources(context.Context, Liquid2SourceSearchRequest) (Liquid2SourceSearchResult, error)
	ReadLiquid2Source(context.Context, Liquid2SourceReadRequest) (Liquid2SourceDocument, error)
}

// Liquid2SourceSearchRequest describes Liquid2 document search criteria.
type Liquid2SourceSearchRequest struct {
	MissionID string
	Query     string
	Limit     int
	Cursor    string
	Filters   Liquid2SourceFilters
}

// Liquid2SourceFilters contains optional Liquid2 search filters.
type Liquid2SourceFilters struct {
	Status         string
	Tag            string
	Kind           string
	RatingMin      int
	IncludeDeleted bool
	IncludeTrash   bool
}

// Liquid2SourceSearchResult contains unapproved Liquid2 source candidates.
type Liquid2SourceSearchResult struct {
	MissionID  string
	Candidates []Liquid2SourceCandidate
	NextCursor string
}

// Liquid2SourceCandidate represents one Liquid2 document as a source candidate.
type Liquid2SourceCandidate struct {
	Connector     sourcecontract.ConnectorRef
	Title         string
	SourceURI     string
	Summary       string
	MatchedRanges []Liquid2MatchedRange
	UpdatedAt     time.Time
	CanSnapshot   bool
}

// Liquid2MatchedRange identifies a matching content range.
type Liquid2MatchedRange struct {
	ContentID string `json:"content_id,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

// Liquid2SourceReadRequest identifies one Liquid2 document.
type Liquid2SourceReadRequest struct {
	ExternalSourceID string
}

// Liquid2SourceDocument contains snapshot-capable Liquid2 content and metadata.
type Liquid2SourceDocument struct {
	Connector sourcecontract.ConnectorRef
	Title     string
	SourceURI string
	UpdatedAt time.Time
	Contents  []Liquid2SourceContent
	Metadata  json.RawMessage
}

// Liquid2SourceContent is one content block in a Liquid2 document.
type Liquid2SourceContent struct {
	ContentID string
	Role      string
	Format    string
	Language  string
	Content   string
}

// SnapshotLiquid2SourceRequest contains the input to persist a Liquid2 source snapshot.
type SnapshotLiquid2SourceRequest struct {
	MissionID           string
	ArtifactID          string
	SnapshotID          string
	ExternalSourceID    string
	Producer            ledger.Producer
	Reason              string
	ContentRanges       []Liquid2ContentRange
	ExpectedContentHash sourcecontract.ContentHash
}

// Liquid2ContentRange selects a content range for a snapshot.
type Liquid2ContentRange struct {
	ContentID string
	Start     int
	End       int
}

// Liquid2SnapshotResult contains the artifact and source snapshot created together.
type Liquid2SnapshotResult struct {
	Artifact artifactcontract.Raw
	Snapshot sourcecontract.Snapshot
}

// SnapshotLiquid2SourceWithEventRequest contains a snapshot request and its event metadata.
type SnapshotLiquid2SourceWithEventRequest struct {
	Snapshot SnapshotLiquid2SourceRequest
	EventID  string
	Producer ledger.Producer
}

// Liquid2SnapshotWithEventResult contains the snapshot artifacts and committed event.
type Liquid2SnapshotWithEventResult struct {
	Artifact artifactcontract.Raw
	Snapshot sourcecontract.Snapshot
	Event    ledger.Event
}
