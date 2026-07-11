package app

import (
	"context"
	"encoding/json"
	"time"
)

const (
	Liquid2ConnectorID        = "liquid2"
	Liquid2ConnectorType      = "liquid2"
	Liquid2HTTPConnectorV1    = "liquid2-http.v1"
	Liquid2SnapshotMediaType  = "application/vnd.plasma.liquid2.snapshot+json"
	Liquid2SnapshotSchemaV1   = "plasma.liquid2.snapshot.v1"
	defaultLiquid2SearchLimit = 10
	maxLiquid2SearchLimit     = 100
)

type Liquid2SourceConnector interface {
	SearchLiquid2Sources(context.Context, Liquid2SourceSearchRequest) (Liquid2SourceSearchResult, error)
	ReadLiquid2Source(context.Context, Liquid2SourceReadRequest) (Liquid2SourceDocument, error)
}

type Liquid2SourceSearchRequest struct {
	MissionID string
	Query     string
	Limit     int
	Cursor    string
	Filters   Liquid2SourceFilters
}

type Liquid2SourceFilters struct {
	Status         string
	Tag            string
	Kind           string
	RatingMin      int
	IncludeDeleted bool
	IncludeTrash   bool
}

type Liquid2SourceSearchResult struct {
	MissionID  string
	Candidates []Liquid2SourceCandidate
	NextCursor string
}

type Liquid2SourceCandidate struct {
	Connector     ConnectorRef
	Title         string
	SourceURI     string
	Summary       string
	MatchedRanges []Liquid2MatchedRange
	UpdatedAt     time.Time
	CanSnapshot   bool
}

type Liquid2MatchedRange struct {
	ContentID string `json:"content_id,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

type Liquid2SourceReadRequest struct {
	ExternalSourceID string
}

type Liquid2SourceDocument struct {
	Connector ConnectorRef
	Title     string
	SourceURI string
	UpdatedAt time.Time
	Contents  []Liquid2SourceContent
	Metadata  json.RawMessage
}

type Liquid2SourceContent struct {
	ContentID string
	Role      string
	Format    string
	Language  string
	Content   string
}

type SnapshotLiquid2SourceRequest struct {
	MissionID           string
	ArtifactID          string
	SnapshotID          string
	ExternalSourceID    string
	Producer            Producer
	Reason              string
	ContentRanges       []Liquid2ContentRange
	ExpectedContentHash ContentHash
}

type Liquid2ContentRange struct {
	ContentID string
	Start     int
	End       int
}

type Liquid2SnapshotResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
}
