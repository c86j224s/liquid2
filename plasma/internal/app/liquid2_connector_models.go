package app

import (
	"context"
	"encoding/json"
	"time"
)

const (
	// Liquid2Connector* 상수는 Liquid2 source connector identity와 snapshot schema를
	// 정의한다.
	Liquid2ConnectorID        = "liquid2"
	Liquid2ConnectorType      = "liquid2"
	Liquid2HTTPConnectorV1    = "liquid2-http.v1"
	Liquid2SnapshotMediaType  = "application/vnd.plasma.liquid2.snapshot+json"
	Liquid2SnapshotSchemaV1   = "plasma.liquid2.snapshot.v1"
	defaultLiquid2SearchLimit = 10
	maxLiquid2SearchLimit     = 100
)

// Liquid2SourceConnector는 Liquid2 문서를 Plasma source 후보와 snapshot 입력으로
// 읽는 connector port다.
type Liquid2SourceConnector interface {
	SearchLiquid2Sources(context.Context, Liquid2SourceSearchRequest) (Liquid2SourceSearchResult, error)
	ReadLiquid2Source(context.Context, Liquid2SourceReadRequest) (Liquid2SourceDocument, error)
}

// Liquid2SourceSearchRequest는 Liquid2 문서 후보 검색 조건이다.
type Liquid2SourceSearchRequest struct {
	MissionID string
	Query     string
	Limit     int
	Cursor    string
	Filters   Liquid2SourceFilters
}

// Liquid2SourceFilters는 Liquid2 검색 API에 전달할 선택적 필터다.
type Liquid2SourceFilters struct {
	Status         string
	Tag            string
	Kind           string
	RatingMin      int
	IncludeDeleted bool
	IncludeTrash   bool
}

// Liquid2SourceSearchResult는 아직 승인되지 않은 Liquid2 source 후보 목록이다.
type Liquid2SourceSearchResult struct {
	MissionID  string
	Candidates []Liquid2SourceCandidate
	NextCursor string
}

// Liquid2SourceCandidate는 Liquid2 문서 하나를 Plasma source 후보로 표현한 값이다.
type Liquid2SourceCandidate struct {
	Connector     ConnectorRef
	Title         string
	SourceURI     string
	Summary       string
	MatchedRanges []Liquid2MatchedRange
	UpdatedAt     time.Time
	CanSnapshot   bool
}

// Liquid2MatchedRange는 Liquid2 검색 결과가 문서 content 안에서 매칭된 범위다.
type Liquid2MatchedRange struct {
	ContentID string `json:"content_id,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

// Liquid2SourceReadRequest는 Liquid2 문서 하나를 읽기 위한 입력이다.
type Liquid2SourceReadRequest struct {
	ExternalSourceID string
}

// Liquid2SourceDocument는 snapshot 가능한 Liquid2 문서 본문과 metadata다.
type Liquid2SourceDocument struct {
	Connector ConnectorRef
	Title     string
	SourceURI string
	UpdatedAt time.Time
	Contents  []Liquid2SourceContent
	Metadata  json.RawMessage
}

// Liquid2SourceContent는 Liquid2 문서 안의 개별 content block이다.
type Liquid2SourceContent struct {
	ContentID string
	Role      string
	Format    string
	Language  string
	Content   string
}

// SnapshotLiquid2SourceRequest는 Liquid2 문서를 mission source snapshot으로 저장하는 입력이다.
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

// Liquid2ContentRange는 snapshot에 포함할 Liquid2 content 범위다.
type Liquid2ContentRange struct {
	ContentID string
	Start     int
	End       int
}

// Liquid2SnapshotResult는 Liquid2 source snapshot 생성 결과다.
type Liquid2SnapshotResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
}
